import argparse
import importlib.util
import ipaddress
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('worker_preview', ROOT / 'scripts/verify-aws-workers.py')
worker = importlib.util.module_from_spec(spec)
spec.loader.exec_module(worker)


def values():
    return {'Stage': 'ready', 'EnableEnvironment': 'true', 'EnvironmentReady': 'true', 'GatewayImageDigest': 'sha256:' + 'b' * 64, 'GatewayCertificateSecretArn': 'arn:aws:secretsmanager:us-east-1:123456789012:secret:gateway/certificate-AbCdEf', 'GatewayKeySecretArn': 'arn:aws:secretsmanager:us-east-1:123456789012:secret:gateway/key-GhIjKl'}


def permission(value, account):
    result = {'IpProtocol': value['IpProtocol'], 'FromPort': value['FromPort'], 'ToPort': value['ToPort']}
    group = value.get('SourceSecurityGroupId', value.get('DestinationSecurityGroupId'))
    if group:
        result['UserIdGroupPairs'] = [{'GroupId': group, 'UserId': account}]
    if 'DestinationPrefixListId' in value:
        result['PrefixListIds'] = [{'PrefixListId': value['DestinationPrefixListId']}]
    if 'CidrIp' in value:
        result['IpRanges'] = [{'CidrIp': value['CidrIp']}]
    return result


def observations(template, preview, settings):
    props = {name: preview.properties(name) for name in template['Resources']}
    refs = preview.references
    account = settings['worker']['account']
    model = {name: [] for name in ['vpcs', 'dhcp', 'subnets', 'groups', 'routes', 'endpoints', 'peers', 'nats', 'internet']}
    kind = lambda name: template['Resources'][name]['Type']
    for name, value in props.items():
        if value is None:
            continue
        if kind(name) == 'AWS::EC2::VPC':
            dhcp = name.removesuffix('Vpc') + 'Dhcp'
            model['vpcs'].append({'VpcId': refs[name], 'OwnerId': account, 'State': 'available', 'CidrBlock': value['CidrBlock'], 'DhcpOptionsId': refs[dhcp]})
        elif kind(name) == 'AWS::EC2::DHCPOptions':
            model['dhcp'].append({'DhcpOptionsId': refs[name], 'OwnerId': account, 'DhcpConfigurations': [{'Key': 'domain-name-servers', 'Values': [{'Value': item} for item in value['DomainNameServers']]}]})
        elif kind(name) == 'AWS::EC2::Subnet':
            model['subnets'].append({'SubnetId': refs[name], 'OwnerId': account, 'State': 'available', **value})
        elif kind(name) == 'AWS::EC2::SecurityGroup':
            group = {'GroupId': refs[name], 'OwnerId': account, 'VpcId': value['VpcId'], 'IpPermissions': [permission(item, account) for item in value.get('SecurityGroupIngress', [])], 'IpPermissionsEgress': [permission(item, account) for item in value.get('SecurityGroupEgress', [])]}
            for other, extra in props.items():
                if extra and extra.get('GroupId') == refs[name] and kind(other) in ['AWS::EC2::SecurityGroupIngress', 'AWS::EC2::SecurityGroupEgress']:
                    group['IpPermissions' if kind(other).endswith('Ingress') else 'IpPermissionsEgress'].append(permission(extra, account))
            model['groups'].append(group)
        elif kind(name) == 'AWS::EC2::RouteTable':
            vpc = next(item for item in model['vpcs'] if item['VpcId'] == value['VpcId'])
            table = {'RouteTableId': refs[name], 'OwnerId': account, 'VpcId': value['VpcId'], 'Associations': [], 'Routes': [{'DestinationCidrBlock': vpc['CidrBlock'], 'GatewayId': 'local', 'State': 'active'}]}
            for other, extra in props.items():
                if not extra:
                    continue
                if kind(other) == 'AWS::EC2::SubnetRouteTableAssociation' and extra['RouteTableId'] == refs[name]:
                    table['Associations'].append({'SubnetId': extra['SubnetId']})
                elif kind(other) == 'AWS::EC2::Route' and extra['RouteTableId'] == refs[name]:
                    table['Routes'].append({**{k: v for k, v in extra.items() if k != 'RouteTableId'}, 'State': 'active'})
                elif kind(other) == 'AWS::EC2::VPCEndpoint' and extra['VpcEndpointType'] == 'Gateway' and refs[name] in extra['RouteTableIds']:
                    table['Routes'].append({'DestinationPrefixListId': settings['network']['s3PrefixList'], 'GatewayId': refs[other], 'State': 'active'})
            model['routes'].append(table)
        elif kind(name) == 'AWS::EC2::VPCEndpoint':
            model['endpoints'].append({'VpcEndpointId': refs[name], 'OwnerId': account, 'State': 'Available', **{k: v for k, v in value.items() if k not in ['SecurityGroupIds', 'PolicyDocument']}, 'Groups': [{'GroupId': group} for group in value.get('SecurityGroupIds', [])], 'NetworkInterfaceIds': ['eni-' + str(index + 1).zfill(17) for index in range(len(value.get('SubnetIds', [])))], **({'PolicyDocument': json.dumps(value['PolicyDocument'], separators=(',', ':'))} if 'PolicyDocument' in value else {})})
        elif kind(name) == 'AWS::EC2::VPCPeeringConnection':
            def info(vpc_id):
                vpc = next(item for item in model['vpcs'] if item['VpcId'] == vpc_id)
                return {'VpcId': vpc_id, 'OwnerId': account, 'Region': settings['worker']['region'], 'CidrBlock': vpc['CidrBlock']}
            model['peers'].append({'VpcPeeringConnectionId': refs[name], 'Status': {'Code': 'active'}, 'RequesterVpcInfo': info(value['VpcId']), 'AccepterVpcInfo': info(value['PeerVpcId'])})
        elif kind(name) == 'AWS::EC2::NatGateway':
            subnet = next(item for item in model['subnets'] if item['SubnetId'] == value['SubnetId'])
            model['nats'].append({'NatGatewayId': refs[name], 'VpcId': subnet['VpcId'], 'SubnetId': value['SubnetId'], 'ConnectivityType': value['ConnectivityType'], 'State': 'available', 'NatGatewayAddresses': [{'AllocationId': value['AllocationId'], 'NetworkInterfaceId': 'eni-12345678901234567', 'PublicIp': '203.0.113.' + str(len(model['nats']) + 2), 'Status': 'succeeded'}]})
        elif kind(name) == 'AWS::EC2::InternetGateway':
            attachments = [extra for other, extra in props.items() if extra and kind(other) == 'AWS::EC2::VPCGatewayAttachment' and extra['InternetGatewayId'] == refs[name]]
            model['internet'].append({'InternetGatewayId': refs[name], 'OwnerId': account, 'Attachments': [{'VpcId': item['VpcId'], 'State': 'available'} for item in attachments]})
    return model


def validate_ranges(preview):
    groups = [['WorkerVpcCidr', 'WorkerSubnetOneCidr', 'WorkerSubnetTwoCidr'], ['EnvironmentWorkerVpcCidr', 'EnvironmentWorkerSubnetOneCidr', 'EnvironmentWorkerSubnetTwoCidr'], ['GatewayVpcCidr', 'GatewaySubnetOneCidr', 'GatewaySubnetTwoCidr', 'GatewayPublicSubnetOneCidr', 'GatewayPublicSubnetTwoCidr']]
    networks = []
    for group in groups:
        parent = ipaddress.ip_network(preview.parameters[group[0]])
        assert parent.is_private and parent.version == 4
        assert not any(parent.overlaps(other) for other in networks)
        networks.append(parent)
        children = [ipaddress.ip_network(preview.parameters[name]) for name in group[1:]]
        assert all(child.subnet_of(parent) for child in children)
        assert all(not left.overlaps(right) for i, left in enumerate(children) for right in children[i + 1:])


def main():
    parser = argparse.ArgumentParser(description='Check the optional ECS environment package locally without cloud access.')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    args.output.mkdir(mode=0o700, parents=True, exist_ok=True)
    lint = subprocess.run([str(Path(sys.executable).with_name('cfn-lint')), '-t', str(ROOT / 'deploy/aws/workers.yaml'), '-r', 'us-east-1'], capture_output=True, text=True, timeout=60)
    (args.output / 'schema.log').write_text(lint.stdout + lint.stderr)
    assert lint.returncode == 0, 'Local template schema check failed'
    template = worker.load('workers.yaml')
    ready = worker.preview(template, values())
    validate_ranges(ready)
    properties = {name: ready.properties(name) for name in template['Resources']}
    settings = json.loads(properties['WorkerSettings']['SecretString'])
    disabled = worker.preview(template, {'Stage': 'ready'})
    assert json.loads(disabled.properties('WorkerSettings')['SecretString']).get('environment') is None
    assert all(disabled.properties(name) is None for name in template['Resources'] if name.startswith(('Environment', 'Gateway')))
    for name, resource in template['Resources'].items():
        if name.startswith(('Environment', 'Gateway')) or name == 'WorkerSettings':
            continue
        assert ready.properties(name) == disabled.properties(name), 'Environment changed offline resource ' + name
    controls = properties['GatewayControllerPolicy']['PolicyDocument']['Statement']
    assert properties['GatewayControllerPolicy']['Roles'] == [ready.parameters['ControllerTaskRoleName']]
    assert all(ready.references['PreparationRole'] not in json.dumps(value) for value in controls)
    assert next(item for item in controls if item['Sid'] == 'PassGatewayPullRole')['Resource'] == settings['environment']['gateway']['executionRole']
    tls = properties['GatewayPullRole']['Policies'][0]['PolicyDocument']['Statement']
    assert next(item for item in tls if item['Action'] == 'secretsmanager:GetSecretValue')['Resource'] == [ready.parameters['GatewayCertificateSecretArn'], ready.parameters['GatewayKeySecretArn']]
    assert 'secretsmanager:' not in json.dumps(properties['WorkerPullRole'])
    assert properties['GatewayRepository']['ImageTagMutability'] == 'IMMUTABLE'
    assert template['Resources']['GatewayRepository']['DeletionPolicy'] == 'Retain'
    assert len(json.dumps(properties['GatewayControllerPolicy']['PolicyDocument'], separators=(',', ':'))) <= 6144
    encrypted = worker.preview(template, {**values(), 'GatewaySecretKmsKeyArn': 'arn:aws:kms:us-east-1:123456789012:key/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee'})
    decrypt = encrypted.properties('GatewayPullRole')['Policies'][0]['PolicyDocument']['Statement'][-1]
    assert decrypt['Action'] == 'kms:Decrypt'
    assert decrypt['Condition']['StringEquals']['kms:ViaService'] == 'secretsmanager.us-east-1.amazonaws.com'
    assert decrypt['Condition']['StringEquals']['kms:EncryptionContext:SecretARN'] == [values()['GatewayCertificateSecretArn'], values()['GatewayKeySecretArn']]
    for stage in ['prepare', 'connect']:
        profile = worker.preview(template, {**values(), 'Stage': stage, 'EnvironmentReady': 'false', 'GatewayImageDigest': ''})
        assert profile.properties('WorkerSettings') is None
        assert (profile.properties('GatewayExchangeEndpoint') is not None) == (stage == 'connect')
    inactive = worker.preview(template, {**values(), 'EnvironmentReady': 'false', 'GatewayImageDigest': ''})
    assert inactive.properties('GatewayRepository') is not None
    assert json.loads(inactive.properties('WorkerSettings')['SecretString'])['environment'] is None
    rejected = [{'GatewayCertificateSecretArn': ''}, {'GatewayKeySecretArn': ''}, {'GatewayKeySecretArn': values()['GatewayCertificateSecretArn']}, {'GatewayImageDigest': ''}, {'Stage': 'connect'}, {'EnableEnvironment': 'false'}]
    for changes in rejected:
        try:
            worker.preview(template, {**values(), **changes})
        except ValueError:
            continue
        raise AssertionError('Incomplete environment parameters accepted')
    fixture = {'settings': settings, 'properties': properties, 'observations': observations(template, ready, settings)}
    (args.output / 'environment-fixture.json').write_text(json.dumps(fixture, indent=2) + '\n')
    summary = {'passed': True, 'cloudOperations': False, 'profiles': 6, 'rejectedConfigurations': len(rejected), 'offlineResourcesUnchanged': True, 'environmentResources': sum(name.startswith(('Environment', 'Gateway')) for name in template['Resources']), 'checks': ['Local schema', 'Default remains offline', 'Existing offline resources unchanged', 'Complete combined settings', 'Narrow controller and certificate roles', 'Scoped optional certificate decryption', 'Separate provisioning and activation', 'Staged endpoint and image prerequisites', 'Private disjoint subnet ranges']}
    (args.output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    print(json.dumps(summary))


if __name__ == '__main__':
    main()
