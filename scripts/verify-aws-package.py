import argparse
import json
import os
from pathlib import Path
import subprocess
import sys

from cfnlint.decode import decode


ROOT = Path(__file__).resolve().parents[1]


def ref(name):
    return {'Ref': name}


def properties(template, name):
    return template['Resources'][name]['Properties']


def main():
    parser = argparse.ArgumentParser(description='Check AWS packaging locally without AWS credentials or service calls.')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    source = ROOT / 'deploy/aws/service.yaml'
    environment = {key: value for key, value in os.environ.items() if not key.startswith(('AWS_', 'AMAZON_'))}
    environment['AWS_EC2_METADATA_DISABLED'] = 'true'
    command = [str(Path(sys.executable).with_name('cfn-lint')), '--template', str(source), '--regions', 'us-east-1', 'us-west-2']
    lint = subprocess.run(command, capture_output=True, text=True, env=environment, timeout=60)
    (output / 'schema.log').write_text(lint.stdout + lint.stderr)
    assert lint.returncode == 0, 'Local CloudFormation schema validation failed; inspect schema.log'
    template, errors = decode(str(source))
    assert not errors
    checks = ['CloudFormation lint and provider schemas pass locally for two regions']
    resource_types = [resource['Type'] for resource in template['Resources'].values()]
    assert resource_types.count('AWS::CloudFront::Distribution') == 2
    ui = properties(template, 'UiDistribution')['DistributionConfig']
    artifact = properties(template, 'ArtifactDistribution')['DistributionConfig']
    assert ui['Aliases'] == [ref('UiHostname')] and artifact['Aliases'] == [ref('ArtifactHostname')]
    assert len(template['Rules']['SeparateOrigins']['Assertions']) == 3
    for distribution in [ui, artifact]:
        assert all(error['ErrorCachingMinTTL'] == 0 and 'ResponsePagePath' not in error and 'ResponseCode' not in error for error in distribution['CustomErrorResponses'])
        assert {error['ErrorCode'] for error in distribution['CustomErrorResponses']} == {400, 403, 404, 405, 414, 416, 500, 501, 502, 503, 504}
    routes = ui['CacheBehaviors']
    assert {r['PathPattern'] for r in routes} == {'/api', '/api/*', '/auth', '/auth/*', '/healthz', '/livez'}
    assert all(r['TargetOriginId'] == 'api' and r['CachePolicyId'] == ref('NoCache') and 'FunctionAssociations' not in r for r in routes)
    assert artifact['DefaultCacheBehavior']['CachePolicyId'] == ref('NoCache')
    assert 'ResponseHeadersPolicyId' not in artifact['DefaultCacheBehavior']
    no_cache = properties(template, 'NoCache')['CachePolicyConfig']
    assert no_cache['MinTTL'] == no_cache['DefaultTTL'] == no_cache['MaxTTL'] == 0
    forwarding = properties(template, 'ForwardRequest')['OriginRequestPolicyConfig']
    assert forwarding['CookiesConfig']['CookieBehavior'] == forwarding['QueryStringsConfig']['QueryStringBehavior'] == 'all'
    assert forwarding['HeadersConfig'] == {'HeaderBehavior': 'allExcept', 'Headers': ['host']}
    checks.append('Private API, identity, and evidence requests cannot use UI fallback or CDN response caching')
    for name in ['UiBucket', 'EvidenceBucket']:
        bucket = properties(template, name)
        assert all(bucket['PublicAccessBlockConfiguration'].values())
        assert bucket['OwnershipControls']['Rules'] == [{'ObjectOwnership': 'BucketOwnerEnforced'}]
        assert 'WebsiteConfiguration' not in bucket and 'LifecycleConfiguration' not in bucket
        assert template['Resources'][name]['DeletionPolicy'] == template['Resources'][name]['UpdateReplacePolicy'] == 'Retain'
    access = properties(template, 'UiOriginAccess')['OriginAccessControlConfig']
    assert access['SigningBehavior'] == 'always' and access['SigningProtocol'] == 'sigv4'
    evidence_policy = properties(template, 'EvidencePolicy')['PolicyDocument']['Statement']
    assert all(rule['Effect'] == 'Deny' for rule in evidence_policy)
    bucket_access = properties(template, 'UiPolicy')['PolicyDocument']['Statement'][0]
    assert bucket_access['Principal'] == {'Service': 'cloudfront.amazonaws.com'} and 'AWS:SourceArn' in bucket_access['Condition']['StringEquals']
    checks.append('Retained private buckets separate distribution-scoped UI access from application-authorized evidence')
    task = properties(template, 'Task')
    container = task['ContainerDefinitions'][0]
    assert container['User'] == '65532:65532' and container['ReadonlyRootFilesystem']
    assert container['LinuxParameters']['Capabilities']['Drop'] == ['ALL']
    assert '@${ServiceImageDigest}' in container['Image']['Fn::Sub']
    assert container['MountPoints'] == [{'SourceVolume': 'temporary', 'ContainerPath': '/tmp', 'ReadOnly': False}]
    assert 'VOLUME ["/tmp"]' in (ROOT / 'deploy/service/Dockerfile').read_text()
    assert '--require-postgres=true' in container['Command']
    assert '--oidc-config-env=TESTSTUDIO_OIDC_CONFIGURATION' in container['Command']
    assert '--artifact-store-env=TESTSTUDIO_ARTIFACT_CONFIGURATION' in container['Command']
    assert '--go-images' not in json.dumps(container) and 'docker.sock' not in json.dumps(template)
    assert container['HealthCheck']['Command'] == ['CMD', '/app/teststudio-health', '--url=http://127.0.0.1:8040/livez']
    assert properties(template, 'ApiTarget')['HealthCheckPath'] == properties(template, 'ArtifactTarget')['HealthCheckPath'] == '/healthz'
    assert properties(template, 'ArtifactTarget')['HealthCheckPort'] == '8040'
    checks.append('Non-root immutable core service keeps writable temporary storage and separates readiness from process liveness')
    secrets = container['Secrets']
    assert secrets[:2] == [{'Name': 'TESTSTUDIO_DATABASE', 'ValueFrom': ref('DatabaseSecretArn')}, {'Name': 'TESTSTUDIO_OIDC_CONFIGURATION', 'ValueFrom': ref('OidcConfigurationSecretArn')}]
    assert all('Default' not in template['Parameters'][name] for name in ['DatabaseSecretArn', 'OidcConfigurationSecretArn', 'BootstrapIssuer', 'BootstrapSubject', 'ServiceRepositoryName', 'ServiceImageDigest'])
    execution = properties(template, 'ExecutionRole')['Policies'][0]['PolicyDocument']['Statement']
    runtime = properties(template, 'TaskRole')['Policies'][0]['PolicyDocument']['Statement']
    assert any(statement.get('Action') == 'secretsmanager:GetSecretValue' for statement in execution)
    assert all('secretsmanager' not in json.dumps(statement) and 'ecr:' not in json.dumps(statement) for statement in runtime)
    assert all('UiBucket' not in json.dumps(statement) for statement in runtime)
    assert runtime[0] == {'Effect': 'Allow', 'Action': 's3:ListBucket', 'Resource': {'Fn::GetAtt': ['EvidenceBucket', 'Arn']}}
    assert runtime[1]['Resource'] == {'Fn::Sub': '${EvidenceBucket.Arn}/teststudio/*'}
    checks.append('Database and two optional OIDC client credentials use startup-only secret references with separate runtime permissions')
    service = properties(template, 'Service')
    assert service['NetworkConfiguration']['AwsvpcConfiguration']['AssignPublicIp'] == 'DISABLED'
    assert service['NetworkConfiguration']['AwsvpcConfiguration']['Subnets'] == ref('PrivateSubnets')
    ingress = properties(template, 'LoadBalancerSecurityGroup')['SecurityGroupIngress']
    assert ingress == [{'IpProtocol': 'tcp', 'FromPort': 443, 'ToPort': 443, 'SourcePrefixListId': ref('CloudFrontPrefixListId')}]
    assert properties(template, 'Listener')['DefaultActions'][0]['FixedResponseConfig']['StatusCode'] == '403'
    for name, target in [('ApiRoute', 'ApiTarget'), ('ArtifactRoute', 'ArtifactTarget')]:
        rule = properties(template, name)
        assert rule['Conditions'][0]['HttpHeaderConfig']['Values'] == [{'Fn::Sub': '{{resolve:secretsmanager:${OriginToken}:SecretString}}'}]
        assert rule['Actions'][0]['TargetGroupArn'] == ref(target)
    assert service['DeploymentConfiguration']['DeploymentCircuitBreaker'] == {'Enable': True, 'Rollback': True}
    checks.append('Private task interfaces and CloudFront-only TLS ingress retain distinct API and artifact listener routes')
    function = properties(template, 'UiRoutes')['FunctionCode']
    script = 'const vm = require("node:vm"); const assert = require("node:assert/strict"); const context = {}; vm.createContext(context); vm.runInContext(' + json.dumps(function) + ', context);\n'
    script += '''for (const [uri, expected] of [['/', '/index.html'], ['/projects/demo/tests', '/index.html'], ['/main-123.js', '/main-123.js'], ['/assets/icons.svg', '/assets/icons.svg']]) {
const request = {uri, method:'GET', querystring:{search:{value:'keep'}}, headers:{}};
const actual = context.handler({request}); assert.equal(actual.uri, expected); assert.equal(actual.querystring.search.value, 'keep'); assert.equal(actual.method, 'GET');
}
'''
    result = subprocess.run(['node', '-e', script], capture_output=True, text=True, timeout=10)
    (output / 'ui-routing.log').write_text(result.stdout + result.stderr)
    assert result.returncode == 0, 'UI routing function check failed'
    checks.append('UI-only routing function preserves asset paths, query strings, and request methods')
    summary = {'passed': True, 'scope': 'local schema and packaging contracts only; no AWS operation', 'resources': len(resource_types), 'regions': ['us-east-1', 'us-west-2'], 'checks': checks}
    (output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    print(json.dumps(summary, indent=2))


if __name__ == '__main__':
    main()
