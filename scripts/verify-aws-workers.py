import argparse
import copy
import json
import os
from pathlib import Path
import subprocess
import sys

from cfnlint.decode import decode
from lib.cloudformation_preview import Preview, identifiers


ROOT = Path(__file__).resolve().parents[1]


def load(name):
    value, errors = decode(str(ROOT / 'deploy/aws' / name))
    assert not errors, errors
    return value


def parameters(region):
    return {'ControllerVpcId': 'vpc-bbbbbbbb', 'ControllerSubnetOne': 'subnet-cccccccc', 'ControllerSubnetTwo': 'subnet-dddddddd', 'AvailabilityZoneOne': region + 'a', 'AvailabilityZoneTwo': region + 'b', 'ControllerClusterArn': f'arn:aws:ecs:{region}:123456789012:cluster/core', 'ControllerTaskRoleName': 'studio-api', 'ControllerSecurityGroupId': 'sg-eeeeeeee', 'DatabaseSecurityGroupId': 'sg-ffffffff', 'S3PrefixListId': 'pl-aaaaaaaa', 'ExchangeHostname': 'workers.studio.internal.example.test', 'ExchangeCertificateArn': f'arn:aws:acm:{region}:123456789012:certificate/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee', 'WorkerFamilyPrefix': 'studio-worker', 'RuntimeRepositoryPrefix': 'teststudio', 'ServiceRepositoryName': 'teststudio/service', 'ServiceImageDigest': 'sha256:' + 'a' * 64, 'DatabaseSecretArn': f'arn:aws:secretsmanager:{region}:123456789012:secret:database-fixture'}


def preview(template, overrides=None, region='us-east-1'):
    references, attributes = identifiers(template, region)
    return Preview(template, {**parameters(region), **(overrides or {})}, references, attributes, region)


def main():
    parser = argparse.ArgumentParser(description='Validate ECS worker packaging locally without credentials, AWS calls, or deployment.')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True, mode=0o700)
    environment = {key: value for key, value in os.environ.items() if not key.startswith(('AWS_', 'AMAZON_'))}
    environment['AWS_EC2_METADATA_DISABLED'] = 'true'
    result = subprocess.run([str(Path(sys.executable).with_name('cfn-lint')), '-t', str(ROOT / 'deploy/aws/service.yaml'), str(ROOT / 'deploy/aws/workers.yaml'), '-r', 'us-east-1', 'us-west-2'], env=environment, capture_output=True, text=True, timeout=60)
    (args.output / 'schema.log').write_text(result.stdout + result.stderr)
    assert result.returncode == 0, 'CloudFormation schema check failed'
    template = load('workers.yaml')
    core = load('service.yaml')
    checks = ['Both CloudFormation templates pass local schemas and dependency checks in two regions']
    default = preview(template)
    assert not any(resource['Type'] in ['AWS::EC2::InternetGateway', 'AWS::EC2::NatGateway', 'AWS::EC2::Route', 'AWS::EC2::VPCPeeringConnection', 'AWS::Lambda::Function'] and default.properties(name) is not None for name, resource in template['Resources'].items())
    snapshots = []
    for region in ['us-east-1', 'us-west-2']:
        for stage in ['prepare', 'connect', 'ready']:
            profile = preview(template, {'Stage': stage}, region)
            ready = stage == 'ready'
            assert profile.properties('ExchangeService')['AcceptanceRequired'] == ready
            assert (profile.properties('ExchangeEndpoint') is not None) == (stage != 'prepare')
            assert (profile.properties('WorkerSettings') is not None) == ready
            assert profile.properties('PreparationService') is None
            snapshots.append({'region': region, 'stage': stage, 'settingsReleased': ready, 'preparationEnabled': False})
    checks.append('Prepare, connect, and ready stages keep settings and new preparation disabled until sealing')
    for overrides in [{'EnablePreparation': 'true'}, {'Stage': 'connect', 'EnablePreparation': 'true'}, {'EnableNewBuilds': 'true'}, {'Stage': 'ready', 'EnablePreparation': 'true', 'EnableNewBuilds': 'true'}, {'Stage': 'unknown'}, {'WorkerFamilyPrefix': 'invalid prefix'}, {'AvailabilityZoneTwo': 'us-east-1a'}, {'ControllerSubnetTwo': 'subnet-cccccccc'}]:
        try:
            preview(template, overrides)
        except ValueError:
            pass
        else:
            raise AssertionError('Invalid worker profile accepted')
    checks.append('Eight invalid stages, runtime choices, and placement inputs fail before deployment')
    ready = preview(template, {'Stage': 'ready', 'EnablePreparation': 'true'})
    settings = json.loads(ready.properties('WorkerSettings')['SecretString'])
    properties = {name: ready.properties(name) for name in template['Resources']}
    fixture = {'settings': settings, 'properties': properties}
    (args.output / 'admission-fixture.json').write_text(json.dumps(fixture, indent=2) + '\n')
    assert properties['WorkerVpc']['EnableDnsSupport'] and properties['WorkerVpc']['EnableDnsHostnames']
    for subnet in ['WorkerSubnetOne', 'WorkerSubnetTwo']:
        assert properties[subnet]['VpcId'] == settings['network']['vpc'] and not properties[subnet]['MapPublicIpOnLaunch'] and not properties[subnet]['AssignIpv6AddressOnCreation'] and not properties[subnet]['EnableDns64']
    assert properties['WorkerDhcp']['DomainNameServers'] == ['AmazonProvidedDNS']
    assert properties['WorkerGroup']['SecurityGroupEgress'] == [{'IpProtocol': 'tcp', 'FromPort': 443, 'ToPort': 443, 'DestinationPrefixListId': settings['network']['s3PrefixList']}]
    assert 'SecurityGroupIngress' not in properties['WorkerGroup']
    for group in ['ExchangeGroup', 'RegistryGroup']:
        assert properties[group]['SecurityGroupIngress'] == [{'IpProtocol': 'tcp', 'FromPort': 443, 'ToPort': 443, 'SourceSecurityGroupId': settings['worker']['securityGroups'][0]}]
        assert properties[group]['SecurityGroupEgress'] == [{'IpProtocol': 'tcp', 'FromPort': 443, 'ToPort': 443, 'DestinationSecurityGroupId': settings['worker']['securityGroups'][0]}]
    assert properties['WorkerExchangeEgress']['DestinationSecurityGroupId'] == settings['network']['exchangeGroup']
    assert properties['WorkerRegistryEgress']['DestinationSecurityGroupId'] == settings['network']['registryGroup']
    checks.append('Untrusted worker routing and explicit groups preserve only registry, layer, and private exchange paths')
    assert properties['ExchangeLoadBalancer']['Scheme'] == 'internal' and properties['ExchangeLoadBalancer']['EnforceSecurityGroupInboundRulesOnPrivateLinkTraffic'] == 'off'
    assert properties['ExchangeListener']['Protocol'] == 'TLS' and properties['ExchangeListener']['Port'] == 443
    assert properties['ExchangeTarget']['Port'] == 8042 and properties['ExchangeTarget']['Matcher']['HttpCode'] == '404'
    attributes = {value['Key']: value['Value'] for value in properties['ExchangeTarget']['TargetGroupAttributes']}
    assert attributes == {'deregistration_delay.timeout_seconds': '30', 'preserve_client_ip.enabled': 'false', 'load_balancing.cross_zone.enabled': 'true', 'deregistration_delay.connection_termination.enabled': 'true', 'proxy_protocol_v2.enabled': 'false', 'target_group_health.dns_failover.minimum_healthy_targets.count': '1', 'target_group_health.dns_failover.minimum_healthy_targets.percentage': 'off', 'target_group_health.unhealthy_state_routing.minimum_healthy_targets.count': '1', 'target_group_health.unhealthy_state_routing.minimum_healthy_targets.percentage': 'off'}
    checks.append('Exchange rollout fixes cross-zone routing, healthy quorum, private client addressing, and bounded connection draining')
    assert properties['ExchangePermissions']['AllowedPrincipals'] == ['arn:aws:iam::123456789012:root']
    assert properties['DnsAssociation']['MutationProtection'] == 'ENABLED' and properties['DnsAssociation']['VpcId'] == settings['network']['vpc']
    assert len(properties['AllowedDomains']['Domains']) == 5 and properties['BlockedDomains']['Domains'] == ['*']
    assert properties['DnsRules']['FirewallRules'][1]['BlockResponse'] == 'NXDOMAIN'
    checks.append('Internal TLS, same-account PrivateLink, exact DNS rules, and protected association match the worker contract')
    for name in ['GoRepository', 'PlaywrightRepository', 'Journals', 'JournalAccessPoint', 'WorkerSettings']:
        assert template['Resources'][name]['DeletionPolicy'] == template['Resources'][name]['UpdateReplacePolicy'] == 'Retain'
    assert properties['Journals']['Encrypted'] and properties['Journals']['BackupPolicy']['Status'] == 'ENABLED'
    assert properties['JournalAccessPoint']['PosixUser'] == {'Uid': '65532', 'Gid': '65532'}
    assert properties['JournalAccessPoint']['RootDirectory']['CreationInfo']['Permissions'] == '0700'
    task = properties['PreparationTask']
    container = task['ContainerDefinitions'][0]
    assert container['User'] == '65532:65532' and container['ReadonlyRootFilesystem'] and container['EntryPoint'] == ['/app/teststudio-prepare']
    assert '--execution-location=ecs' in container['Command'] and '--ecs-workers-env=TESTSTUDIO_ECS_CONFIGURATION' in container['Command']
    assert not any(argument.startswith('--registry-') for argument in container['Command'])
    assert 'PortMappings' not in container and 'LoadBalancers' not in properties['PreparationService']
    volume = task['Volumes'][1]['EFSVolumeConfiguration']
    assert volume['TransitEncryption'] == 'ENABLED' and volume['AuthorizationConfig']['IAM'] == 'ENABLED'
    assert properties['PreparationService']['NetworkConfiguration']['AwsvpcConfiguration']['AssignPublicIp'] == 'DISABLED'
    enabled = preview(template, {'Stage': 'ready', 'EnablePreparation': 'true', 'EnableNewBuilds': 'true', 'GoRuntimeDigest': 'sha256:' + 'b' * 64})
    command = enabled.properties('PreparationTask')['ContainerDefinitions'][0]['Command']
    assert any(value.startswith('--registry-go-runtime=') for value in command) and not any(value.startswith('--registry-playwright-runtime=') for value in command)
    checks.append('Separate preparation identity, private persistent journals, one-runner selection, and cleanup-only mode remain distinct from untrusted tasks')
    policies = properties['ControllerPolicy']['PolicyDocument']['Statement']
    by_name = {value['Sid']: value for value in policies}
    assert any('elasticloadbalancing:DescribeTargetGroupAttributes' in value.get('Action', []) for value in policies)
    assert by_name['LaunchOwnedTasks']['Condition']['ArnEquals']['ecs:cluster'] == settings['worker']['cluster']
    assert by_name['PassPlatformRole']['Resource'] == settings['worker']['executionRole']
    assert by_name['PassDeadlineRole']['Resource'] == settings['worker']['deadlineRole']
    assert by_name['PassPlatformRole']['Condition']['StringEquals']['iam:PassedToService'] == 'ecs-tasks.amazonaws.com'
    assert by_name['PassDeadlineRole']['Condition']['StringEquals']['iam:PassedToService'] == 'scheduler.amazonaws.com'
    assert properties['DeadlineRole']['Policies'][0]['PolicyDocument']['Statement'][0]['Action'] == 'ecs:StopTask'
    assert 'secretsmanager:' not in json.dumps(properties['WorkerPullRole']) and 'elasticfilesystem:' not in json.dumps(properties['WorkerPullRole'])
    assert len(json.dumps(properties['ControllerPolicy']['PolicyDocument'], separators=(',', ':'))) <= 6144
    assert all('ecr:PutImage' not in json.dumps(value) for value in policies)
    checks.append('Control, image pull, registry publication, filesystem, and independent stop permissions remain separated and scoped')
    refs, attrs = identifiers(core)
    refs.update({'OriginToken': 'private-origin-token-reference', 'UiBucket': 'studio-ui', 'EvidenceBucket': 'studio-evidence'})
    attrs.update({'EvidenceBucket.Arn': 'arn:aws:s3:::studio-evidence', 'ServiceLog.Arn': 'arn:aws:logs:us-east-1:123456789012:log-group:studio:*'})
    core_parameters = {'VpcId': 'vpc-bbbbbbbb', 'PublicSubnets': ['subnet-eeeeeeee', 'subnet-ffffffff'], 'PrivateSubnets': ['subnet-cccccccc', 'subnet-dddddddd'], 'HostedZoneId': 'ZFIXTURE', 'UiHostname': 'studio.example.test', 'ArtifactHostname': 'evidence.example.test', 'OriginHostname': 'origin.example.test', 'ViewerCertificateArn': 'arn:aws:acm:us-east-1:123456789012:certificate/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee', 'OriginCertificateArn': 'arn:aws:acm:us-east-1:123456789012:certificate/bbbbbbbb-bbbb-cccc-dddd-eeeeeeeeeeee', 'CloudFrontPrefixListId': 'pl-bbbbbbbb', 'ServiceRepositoryName': 'teststudio/service', 'ServiceImageDigest': 'sha256:' + 'a' * 64, 'DatabaseSecretArn': parameters('us-east-1')['DatabaseSecretArn'], 'OidcConfigurationSecretArn': 'arn:aws:secretsmanager:us-east-1:123456789012:secret:oidc-fixture', 'BootstrapIssuer': 'https://identity.example.test', 'BootstrapSubject': 'owner'}
    for mode in ['disabled', 'recovery', 'new-sources']:
        values = copy.deepcopy(core_parameters)
        if mode != 'disabled':
            values.update({'WorkerSettingsSecretArn': ready.references['WorkerSettings'], 'WorkerTargetGroupArn': ready.references['ExchangeTarget'], 'WorkerNewSources': 'true' if mode == 'new-sources' else 'false', 'WorkerSourceRunners': 'go'})
        api = Preview(core, values, refs, attrs)
        container = api.properties('Task')['ContainerDefinitions'][0]
        targets = api.properties('Service')['LoadBalancers']
        assert any(value == '--ecs-workers-env=TESTSTUDIO_ECS_CONFIGURATION' for value in container['Command']) == (mode != 'disabled')
        assert ('--source-bundles' in container['Command']) == (mode == 'new-sources')
        assert ('--prepared-sources' in container['Command']) == (mode == 'new-sources')
        assert len(targets) == (2 if mode == 'disabled' else 3)
        if mode != 'disabled':
            assert targets[2]['ContainerPort'] == 8042 and targets[2]['TargetGroupArn'] == settings['network']['targetGroup']
            assert any(secret['Name'] == 'TESTSTUDIO_ECS_CONFIGURATION' for secret in container['Secrets'])
            assert '--execution-location=ecs' in container['Command']
    checks.append('Core service separates disabled, recovery-only, and new-source settings while preserving public API and evidence targets')
    for overrides in [{'WorkerSettingsSecretArn': ready.references['WorkerSettings']}, {'WorkerTargetGroupArn': ready.references['ExchangeTarget']}, {'WorkerNewSources': 'true'}]:
        try:
            Preview(core, {**core_parameters, **overrides}, refs, attrs)
        except ValueError:
            pass
        else:
            raise AssertionError('Partial core worker configuration accepted')
    checks.append('Three incomplete core configurations fail before deployment')
    (args.output / 'profiles.json').write_text(json.dumps(snapshots, indent=2) + '\n')
    summary = {'passed': True, 'cloudOperations': False, 'credentials': False, 'workerResources': sum(ready.properties(name) is not None for name in template['Resources']), 'templateResources': len(template['Resources']), 'coreResources': len(core['Resources']), 'profiles': len(snapshots) + 3, 'rejectedConfigurations': 11, 'checks': checks}
    (args.output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    print(json.dumps(summary, indent=2))


if __name__ == '__main__':
    main()
