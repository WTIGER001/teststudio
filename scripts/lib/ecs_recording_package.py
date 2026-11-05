import copy
import json

from .cloudformation_preview import Preview, identifiers


def check_recording(template, output):
    assert template['Parameters']['RecordingCompanion']['Default'] == 'false'
    references, attributes = identifiers(template)
    settings = {
        'VpcId': 'vpc-aaaaaaaa', 'PublicSubnets': ['subnet-bbbbbbbb', 'subnet-cccccccc'], 'PrivateSubnets': ['subnet-dddddddd', 'subnet-eeeeeeee'],
        'HostedZoneId': 'ZFIXTURE', 'UiHostname': 'studio.example.test', 'ArtifactHostname': 'evidence.example.test', 'OriginHostname': 'origin.example.test',
        'ViewerCertificateArn': 'arn:aws:acm:us-east-1:123456789012:certificate/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee',
        'OriginCertificateArn': 'arn:aws:acm:us-east-1:123456789012:certificate/bbbbbbbb-bbbb-cccc-dddd-eeeeeeeeeeee',
        'CloudFrontPrefixListId': 'pl-aaaaaaaa', 'ServiceRepositoryName': 'teststudio/service', 'ServiceImageDigest': 'sha256:' + 'a' * 64,
        'DatabaseSecretArn': 'arn:aws:secretsmanager:us-east-1:123456789012:secret:database-fixture',
        'OidcConfigurationSecretArn': 'arn:aws:secretsmanager:us-east-1:123456789012:secret:oidc-fixture',
        'BootstrapIssuer': 'https://identity.example.test', 'BootstrapSubject': 'owner',
    }
    profiles = []
    for workers in [False, True]:
        values = copy.deepcopy(settings)
        if workers:
            values.update(WorkerSettingsSecretArn='arn:aws:secretsmanager:us-east-1:123456789012:secret:worker-fixture', WorkerTargetGroupArn='arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/worker/aaaaaaaaaaaaaaaa')
        disabled = Preview(template, values, references, attributes).properties('Task')
        assert not any('recording' in arg for arg in disabled['ContainerDefinitions'][0]['Command'])
        enabled = Preview(template, {**values, 'RecordingCompanion': 'true'}, references, attributes).properties('Task')
        command = enabled['ContainerDefinitions'][0]['Command']
        assert command.count('--recording-companion=true') == 1
        assert not any('recording-cli' in arg or 'recording-helper' in arg or arg == '--local' for arg in command)
        profiles.extend([{'workers': workers, 'enabled': False, 'task': disabled}, {'workers': workers, 'enabled': True, 'task': copy.deepcopy(enabled)}])
        command.remove('--recording-companion=true')
        assert disabled == enabled
    rejected = []
    for value in ['yes', '', True]:
        try:
            Preview(template, {**settings, 'RecordingCompanion': value}, references, attributes)
        except ValueError:
            rejected.append(str(value))
        else:
            raise AssertionError('Invalid recording enable value accepted')
    distribution = template['Resources']['UiDistribution']['Properties']['DistributionConfig']
    route = next(item for item in distribution['CacheBehaviors'] if item['PathPattern'] == '/api/*')
    assert route['TargetOriginId'] == 'api' and route['ViewerProtocolPolicy'] == 'https-only' and 'POST' in route['AllowedMethods']
    assert route['CachePolicyId'] == {'Ref': 'NoCache'} and route['OriginRequestPolicyId'] == {'Ref': 'ForwardRequest'} and 'FunctionAssociations' not in route
    forwarding = template['Resources']['ForwardRequest']['Properties']['OriginRequestPolicyConfig']
    assert forwarding['HeadersConfig'] == {'HeaderBehavior': 'allExcept', 'Headers': ['host']}
    cache = template['Resources']['NoCache']['Properties']['CachePolicyConfig']
    assert cache['MinTTL'] == cache['DefaultTTL'] == cache['MaxTTL'] == 0
    (output / 'recording-profiles.json').write_text(json.dumps(profiles, indent=2) + '\n')
    return {'profiles': len(profiles), 'rejectedConfigurations': len(rejected), 'defaultDisabled': True, 'taskChanges': 'API argument only', 'httpsRouting': True, 'authorizationForwarded': True, 'responseCaching': False}
