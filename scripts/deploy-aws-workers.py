import argparse
import hashlib
import json
from pathlib import Path
import re
import sys
from datetime import datetime, timezone

import boto3
from botocore.exceptions import ClientError
from cfnlint.decode import decode
from lib.cloudformation_preview import Preview, identifiers


ROOT = Path(__file__).resolve().parents[1]
OWNERSHIP = {'Key': 'Application', 'Value': 'TestStudio'}
COMPLETE = {'CREATE_COMPLETE', 'UPDATE_COMPLETE'}


def require(condition, message):
    if not condition:
        raise ValueError(message)


def digest(value):
    return hashlib.sha256(value).hexdigest()


def write(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.touch(mode=0o600, exist_ok=True)
    path.chmod(0o600)
    path.write_text(json.dumps(value, indent=2) + '\n')


def values(items, key, value):
    return {item[key]: item[value] for item in items}


def find_stack(cf, name):
    try:
        return cf.describe_stacks(StackName=name)['Stacks'][0]
    except ClientError as error:
        if error.response['Error']['Code'] == 'ValidationError' and 'does not exist' in error.response['Error']['Message']:
            return None
        raise


def core_state(core):
    require(OWNERSHIP in core.get('Tags', []), 'Core ownership is required')
    params = values(core['Parameters'], 'ParameterKey', 'ParameterValue')
    outputs = values(core['Outputs'], 'OutputKey', 'OutputValue')
    require(outputs['ServiceName'] == core['StackName'], 'Unexpected core service')
    require(not params.get('WorkerSettingsSecretArn') and params.get('WorkerNewSources') == 'false', 'Preserve an installed worker configuration')
    return {'id': core['StackId'], 'updated': str(core.get('LastUpdatedTime', core['CreationTime'])), 'parameters': {key: params[key] for key in ['VpcId', 'PrivateSubnets', 'DatabaseSecurityGroupId', 'DatabaseSecretArn', 'ServiceRepositoryName', 'ServiceImageDigest', 'Architecture']}, 'outputs': {key: outputs[key] for key in ['TaskRoleName', 'TaskSecurityGroupId', 'ClusterArn', 'ServiceName']}}


def service_state(ecs, core):
    cluster = core['outputs']['ClusterArn']
    services = [arn for page in ecs.get_paginator('list_services').paginate(cluster=cluster) for arn in page['serviceArns']]
    result = {}
    for start in range(0, len(services), 10):
        response = ecs.describe_services(cluster=cluster, services=services[start:start + 10])
        require(not response.get('failures'), 'Inspect missing core or preserved service')
        for service in response['services']:
            require(service['status'] == 'ACTIVE' and service['runningCount'] == service['desiredCount'] and service['pendingCount'] == 0, 'Preserve an unsettled service')
            require(len(service['deployments']) == 1 and service['deployments'][0].get('rolloutState') == 'COMPLETED', 'Inspect an in-progress service rollout')
            result[service['serviceName']] = {'deployment': service['deployments'][0]['id'], 'desired': service['desiredCount']}
    require(core['outputs']['ServiceName'] in result, 'Inspect the missing core service')
    return result


def template_for(phase):
    path = ROOT / 'deploy/aws' / ('worker-bootstrap.yaml' if phase == 'bootstrap' else 'workers.yaml')
    template, errors = decode(str(path))
    require(not errors, 'Invalid CloudFormation template')
    return path.read_bytes(), template


def expected_resources(template, parameters, region):
    refs, attrs = identifiers(template, region)
    typed = {key: int(value) if template['Parameters'][key]['Type'] == 'Number' else value for key, value in parameters.items()}
    preview = Preview(template, typed, refs, attrs, region)
    return {name: resource['Type'] for name, resource in template['Resources'].items() if preview.properties(name) is not None}


def validate_change(change, receipt):
    require(change['Status'] == 'CREATE_COMPLETE' and change['ExecutionStatus'] == 'AVAILABLE', 'The change set is not available; inspect its status')
    require(change['StackName'] == receipt['stack'] and change['ChangeSetId'] == receipt['changeSet'] and change['StackId'] == receipt['stackId'], 'Change-set identity changed')
    if receipt['phase'] == 'restore':
        require(not change.get('Tags'), 'An import must preserve existing resource tags')
    else:
        require(OWNERSHIP in change.get('Tags', []), 'Change-set ownership is required')
    require(values(change['Parameters'], 'ParameterKey', 'ParameterValue') == receipt['parameters'], 'Change-set parameters changed')
    summary = []
    tag_updates = receipt.get('tagUpdates', {})
    for item in change.get('Changes', []):
        resource = item['ResourceChange']
        action = 'Import' if receipt['phase'] == 'restore' else 'Add'
        if resource['Action'] == 'Modify':
            require(resource['LogicalResourceId'] in tag_updates and resource.get('PhysicalResourceId') == tag_updates[resource['LogicalResourceId']], 'Only captured imported resources may receive ownership tags')
            require(resource.get('Replacement') == 'False' and resource.get('Scope') == ['Tags'] and resource.get('Details') and all(detail['Target']['Attribute'] == 'Tags' and detail['Target']['RequiresRecreation'] == 'Never' for detail in resource['Details']), 'Imported resource updates may change only tags without replacement')
        else:
            require(resource['Action'] == action and resource.get('Replacement', 'False') == 'False', 'Initial stages may only add owned resources or restore retained resources')
        summary.append({key: resource[key] for key in ['Action', 'LogicalResourceId', 'ResourceType', 'PhysicalResourceId', 'Scope'] if key in resource})
    expected = receipt.get('changeResources', receipt['resources'])
    require({v['LogicalResourceId']: v['ResourceType'] for v in summary if v['Action'] != 'Modify'} == expected and {v['LogicalResourceId'] for v in summary if v['Action'] == 'Modify'} == set(tag_updates) and len(summary) == len(expected) + len(tag_updates), 'Unexpected resources in the change set')
    return sorted(summary, key=lambda item: item['LogicalResourceId'])


def bootstrap_parameters(session, args):
    require(args.hostname and args.zone_id, 'Bootstrap prepare requires --hostname and --zone-id')
    zone = session.client('route53').get_hosted_zone(Id=args.zone_id)['HostedZone']
    require(not zone['Config']['PrivateZone'], 'Certificate validation requires the public authoritative zone')
    require(args.hostname.startswith(args.core_stack + '-workers.') and args.hostname.endswith('.' + zone['Name'].rstrip('.')), 'Use a dedicated TestStudio worker hostname in the selected zone')
    return {'ExchangeHostname': args.hostname, 'HostedZoneId': args.zone_id.removeprefix('/hostedzone/')}


def worker_parameters(session, args, core, cf):
    bootstrap = find_stack(cf, args.stack + '-bootstrap')
    require(bootstrap and bootstrap['StackStatus'] in COMPLETE and OWNERSHIP in bootstrap.get('Tags', []), 'Complete the owned bootstrap stack first')
    outputs = values(bootstrap['Outputs'], 'OutputKey', 'OutputValue')
    certificate = session.client('acm').describe_certificate(CertificateArn=outputs['ExchangeCertificateArn'])['Certificate']
    require(certificate['Status'] == 'ISSUED' and certificate['DomainName'] == outputs['ExchangeHostname'], 'The exchange certificate is not ready')
    params = core['parameters']
    subnets = params['PrivateSubnets'].split(',')
    require(len(subnets) == 2, 'Select exactly two private controller subnets')
    ec2 = session.client('ec2')
    actual = {v['SubnetId']: v for v in ec2.describe_subnets(SubnetIds=subnets)['Subnets']}
    require(all(actual[v]['VpcId'] == params['VpcId'] and not actual[v]['MapPublicIpOnLaunch'] for v in subnets), 'Controller subnets must remain private in the selected VPC')
    require(actual[subnets[0]]['AvailabilityZone'] != actual[subnets[1]]['AvailabilityZone'], 'Controller subnets need separate availability zones')
    prefix = ec2.describe_managed_prefix_lists(Filters=[{'Name': 'prefix-list-name', 'Values': ['com.amazonaws.' + args.region + '.s3']}])['PrefixLists']
    require(len(prefix) == 1 and prefix[0]['OwnerId'] == 'AWS', 'Select the regional AWS S3 prefix list')
    require(params['ServiceRepositoryName'].startswith(args.core_stack + '/'), 'Preserve service repository ownership')
    return {'Stage': 'prepare', 'ControllerVpcId': params['VpcId'], 'ControllerSubnetOne': subnets[0], 'ControllerSubnetTwo': subnets[1], 'AvailabilityZoneOne': actual[subnets[0]]['AvailabilityZone'], 'AvailabilityZoneTwo': actual[subnets[1]]['AvailabilityZone'], 'ControllerClusterArn': core['outputs']['ClusterArn'], 'ControllerTaskRoleName': core['outputs']['TaskRoleName'], 'ControllerSecurityGroupId': core['outputs']['TaskSecurityGroupId'], 'DatabaseSecurityGroupId': params['DatabaseSecurityGroupId'], 'DatabaseSecretArn': params['DatabaseSecretArn'], 'S3PrefixListId': prefix[0]['PrefixListId'], 'ExchangeHostname': outputs['ExchangeHostname'], 'ExchangeCertificateArn': outputs['ExchangeCertificateArn'], 'WorkerFamilyPrefix': args.stack, 'RuntimeRepositoryPrefix': args.core_stack + '/runtimes', 'ServiceRepositoryName': params['ServiceRepositoryName'], 'ServiceImageDigest': params['ServiceImageDigest'], 'ServiceArchitecture': params['Architecture'], 'WorkerArchitecture': 'arm64' if params['Architecture'] == 'ARM64' else 'amd64', 'EnablePreparation': 'false', 'EnableNewBuilds': 'false', 'EnableEnvironment': 'false'}, outputs['PackageBucket']


def verify_stage(session, stack, receipt):
    require(stack['StackStatus'] in COMPLETE, 'Wait for a completed stack before verification')
    require(OWNERSHIP in stack.get('Tags', []), 'Stack ownership is required')
    require(values(stack['Parameters'], 'ParameterKey', 'ParameterValue') == receipt['parameters'], 'Deployed parameters changed')
    resources = session.client('cloudformation').describe_stack_resources(StackName=stack['StackId'])['StackResources']
    require({item['LogicalResourceId']: item['ResourceType'] for item in resources} == receipt['resources'], 'Deployed resources differ from the reviewed stage')
    require(all(item['ResourceStatus'] in COMPLETE for item in resources), 'Inspect incomplete resources')
    actual = {item['LogicalResourceId']: item['PhysicalResourceId'] for item in resources}
    outputs = values(stack['Outputs'], 'OutputKey', 'OutputValue')
    checks = ['Deployed parameters and exact resource additions match the reviewed stage', 'Core inputs and both completed ECS service deployments remain unchanged']
    if receipt['phase'] == 'bootstrap':
        bucket = outputs['PackageBucket']
        s3 = session.client('s3')
        block = s3.get_public_access_block(Bucket=bucket)['PublicAccessBlockConfiguration']
        require(all(block.values()), 'Package storage must block public access')
        encryption = s3.get_bucket_encryption(Bucket=bucket)['ServerSideEncryptionConfiguration']['Rules']
        require(encryption[0]['ApplyServerSideEncryptionByDefault']['SSEAlgorithm'] == 'AES256', 'Package encryption changed')
        require(s3.get_bucket_versioning(Bucket=bucket).get('Status') == 'Enabled', 'Package versioning changed')
        certificate = session.client('acm').describe_certificate(CertificateArn=outputs['ExchangeCertificateArn'])['Certificate']
        require(certificate['Status'] == 'ISSUED' and certificate['DomainName'] == outputs['ExchangeHostname'], 'The dedicated certificate is not issued')
        checks.extend(['Package storage blocks public access and uses encryption and versioning', 'The dedicated regional exchange certificate is issued'])
        return checks
    require(outputs['Stage'] == 'prepare' and 'WorkerSettingsSecretArn' not in outputs and 'ExchangeEndpointId' not in outputs, 'The initial stage must not release settings or connect workers')
    require(not {'PreparationService', 'PreparationTask', 'WorkerSettings', 'ExchangeEndpoint'} & actual.keys(), 'The initial stage must not enable preparation or execution')
    ecs = session.client('ecs')
    cluster = outputs['WorkerClusterArn']
    require(not ecs.list_tasks(cluster=cluster)['taskArns'] and not ecs.list_services(cluster=cluster)['serviceArns'], 'No worker workload may run during prepare')
    checks.append('No preparation service, worker task, settings secret, or exchange consumer is enabled')
    ec2 = session.client('ec2')
    routes = ec2.describe_route_tables(Filters=[{'Name': 'vpc-id', 'Values': [outputs['WorkerVpcId']]}])['RouteTables']
    require(all(route.get('GatewayId') in {'local', actual['ImageLayers']} for table in routes for route in table['Routes']), 'Worker routes must have no internet or peering path')
    group = ec2.describe_security_groups(GroupIds=[actual['WorkerGroup']])['SecurityGroups'][0]
    require(not group['IpPermissions'], 'The worker group must have no ingress')
    destinations = set()
    for rule in group['IpPermissionsEgress']:
        require(rule['IpProtocol'] == 'tcp' and rule['FromPort'] == rule['ToPort'] == 443 and not rule.get('IpRanges') and not rule.get('Ipv6Ranges'), 'Worker egress must be narrow HTTPS')
        destinations.update(item['GroupId'] for item in rule.get('UserIdGroupPairs', []))
        destinations.update(item['PrefixListId'] for item in rule.get('PrefixListIds', []))
    require(destinations == {actual['ExchangeGroup'], actual['RegistryGroup'], receipt['parameters']['S3PrefixListId']}, 'Worker destinations changed')
    checks.append('Worker routes have no internet path and worker HTTPS egress is restricted to the exchange, registry, and S3')
    elb = session.client('elbv2')
    lb = elb.describe_load_balancers(LoadBalancerArns=[actual['ExchangeLoadBalancer']])['LoadBalancers'][0]
    require(lb['Scheme'] == 'internal' and lb['State']['Code'] == 'active', 'The exchange must remain private and active')
    listeners = elb.describe_listeners(LoadBalancerArn=lb['LoadBalancerArn'])['Listeners']
    require(len(listeners) == 1 and listeners[0]['Protocol'] == 'TLS' and listeners[0]['Port'] == 443 and listeners[0]['Certificates'][0]['CertificateArn'] == receipt['parameters']['ExchangeCertificateArn'], 'The exchange TLS listener changed')
    require(not elb.describe_target_health(TargetGroupArn=actual['ExchangeTarget'])['TargetHealthDescriptions'], 'The initial exchange must have no controller target')
    principals = ec2.describe_vpc_endpoint_service_permissions(ServiceId=outputs['ExchangeServiceId'])['AllowedPrincipals']
    require({item['Principal'] for item in principals} == {'arn:aws:iam::' + receipt['account'] + ':root'}, 'PrivateLink must permit only the installation account')
    checks.append('The private TLS exchange has its dedicated certificate, no controller targets, and only same-account PrivateLink permission')
    efs = session.client('efs')
    fs = efs.describe_file_systems(FileSystemId=outputs['JournalFileSystemId'])['FileSystems'][0]
    require(fs['Encrypted'] and fs['LifeCycleState'] == 'available', 'Private journals must be encrypted and available')
    access = efs.describe_access_points(FileSystemId=fs['FileSystemId'])['AccessPoints']
    require(len(access) == 1 and access[0]['PosixUser'] == {'Uid': 65532, 'Gid': 65532} and access[0]['RootDirectory']['CreationInfo']['Permissions'] == '0700', 'Journal access-point ownership changed')
    mounts = efs.describe_mount_targets(FileSystemId=fs['FileSystemId'])['MountTargets']
    require(len(mounts) == 2 and all(item['LifeCycleState'] == 'available' for item in mounts), 'Both private journal mount targets must be available')
    checks.append('Encrypted journals and both mount targets are available with the dedicated non-root private access point')
    repositories = session.client('ecr').describe_repositories(repositoryNames=[actual['GoRepository'], actual['PlaywrightRepository']])['repositories']
    require(all(item['imageTagMutability'] == 'IMMUTABLE' and item['encryptionConfiguration']['encryptionType'] == 'AES256' for item in repositories), 'Runtime repositories must remain immutable and encrypted')
    checks.append('Both runtime repositories retain immutable tags and encryption')
    identities = session.client('iam').list_entities_for_policy(PolicyArn=actual['ControllerPolicy'])
    require({item['RoleName'] for item in identities['PolicyRoles']} == {receipt['parameters']['ControllerTaskRoleName'], actual['PreparationRole']} and not identities['PolicyUsers'] and not identities['PolicyGroups'] and not identities.get('IsTruncated'), 'Worker controller authority must bind only the API and preparation roles')
    checks.append('Worker control authority is attached only to the selected API and preparation roles')
    return checks


def main():
    parser = argparse.ArgumentParser(description='Prepare and review the initial TestStudio worker infrastructure without enabling execution.')
    parser.add_argument('action', choices=['prepare', 'review', 'execute', 'status', 'verify'])
    parser.add_argument('--phase', choices=['bootstrap', 'workers'], required=True)
    for name in ['profile', 'region', 'account', 'core-stack', 'stack']:
        parser.add_argument('--' + name, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--hostname')
    parser.add_argument('--zone-id')
    args = parser.parse_args()
    require(re.fullmatch(r'[a-z][a-z0-9-]{0,21}', args.stack) and args.stack == args.core_stack + '-workers', 'Use the dedicated core-stack workers name, at most 22 characters')
    session = boto3.Session(profile_name=args.profile, region_name=args.region)
    require(session.client('sts').get_caller_identity()['Account'] == args.account, 'Unexpected AWS account')
    cf = session.client('cloudformation')
    core = find_stack(cf, args.core_stack)
    require(core is not None, 'The core stack does not exist')
    state = core_state(core)
    name = args.stack + ('-bootstrap' if args.phase == 'bootstrap' else '')
    receipt_path = args.output / (args.phase + '.json')
    receipt = json.loads(receipt_path.read_text()) if receipt_path.exists() else None
    restored_path = args.output / 'restore.json'
    restored = json.loads(restored_path.read_text()) if restored_path.exists() else None
    stack = find_stack(cf, name)
    if receipt:
        require(all(receipt[key] == value for key, value in {'account': args.account, 'region': args.region, 'stack': name, 'phase': args.phase}.items()), 'Receipt belongs to another deployment')
    if args.action == 'status':
        require(stack is not None, 'Prepare this phase first')
        if stack['StackStatus'] != 'REVIEW_IN_PROGRESS':
            require(OWNERSHIP in stack.get('Tags', []), 'Stack ownership is required')
        result = {'stack': name, 'status': stack['StackStatus'], 'outputs': stack.get('Outputs', [])}
        if stack['StackStatus'] == 'REVIEW_IN_PROGRESS':
            require(receipt is not None and receipt['stackId'] == stack['StackId'], 'Inspect the existing pending deployment')
            change = cf.describe_change_set(ChangeSetName=receipt['changeSet'])
            result['changeSetStatus'] = change['Status']
            result['reason'] = change.get('StatusReason')
        if receipt:
            receipt['status'] = result
            write(receipt_path, receipt)
        print(json.dumps(result))
        return
    require(core['StackStatus'] in COMPLETE, 'Preserve the core deployment in progress')
    services = service_state(session.client('ecs'), state)
    body, template = template_for(args.phase)
    if args.action == 'verify':
        require(receipt is not None and stack is not None and stack['StackId'] == receipt['stackId'], 'Verify the owned deployment receipt')
        require(state == receipt['core'] and services == receipt['services'], 'Core or preserved service changed')
        require(digest(body) == receipt['templateSha256'], 'The verified template must match the reviewed source')
        checks = verify_stage(session, stack, receipt)
        receipt['verification'] = {'passed': True, 'checks': checks, 'verifiedAt': datetime.now(timezone.utc).isoformat(), 'executionEnabled': False}
        write(receipt_path, receipt)
        print(json.dumps(receipt['verification'], indent=2))
        return
    if args.action == 'prepare':
        if receipt:
            require(receipt['core'] == state and receipt['templateSha256'] == digest(body), 'Inputs changed; inspect the existing receipt')
            if args.phase == 'workers' and restored and restored.get('importComplete') and receipt['stackId'] == restored['stackId']:
                require(stack and stack['StackStatus'] == 'IMPORT_COMPLETE', 'Preserve the started deployment')
                require(receipt['changeResources'] == {key: value for key, value in receipt['resources'].items() if key not in restored['resources']}, 'Imported resource scope changed')
                receipt['tagUpdates'] = restored['retained']
                write(receipt_path, receipt)
            print(json.dumps({'reusedChangeSet': receipt['changeSet']}))
            return
        if stack is not None:
            require(args.phase == 'workers' and restored and restored.get('importComplete') and stack['StackId'] == restored['stackId'] and stack['StackStatus'] == 'IMPORT_COMPLETE', 'Preserve and inspect the existing worker deployment')
        if args.phase == 'bootstrap':
            supplied, bucket = bootstrap_parameters(session, args), None
        else:
            supplied, bucket = worker_parameters(session, args, state, cf)
        params = {key: str(value['Default']) for key, value in template.get('Parameters', {}).items() if 'Default' in value}
        params.update(supplied)
        resources = expected_resources(template, params, args.region)
        change_resources = {key: value for key, value in resources.items() if not stack or key not in restored['resources']}
        request = {'StackName': name, 'ChangeSetName': 'initial-' + args.phase + '-' + digest(body)[:12], 'ChangeSetType': 'UPDATE' if stack else 'CREATE', 'Parameters': [{'ParameterKey': key, 'ParameterValue': value} for key, value in params.items()], 'Tags': [OWNERSHIP], 'Capabilities': ['CAPABILITY_IAM']}
        package = None
        if bucket:
            key = 'worker-templates/' + digest(body) + '.yaml'
            session.client('s3').put_object(Bucket=bucket, Key=key, Body=body, ServerSideEncryption='AES256', ContentType='application/yaml')
            request['TemplateURL'] = 'https://' + bucket + '.s3.' + args.region + '.amazonaws.com/' + key
            package = {'bucket': bucket, 'key': key}
        else:
            request['TemplateBody'] = body.decode()
        response = cf.create_change_set(**request)
        receipt = {'account': args.account, 'region': args.region, 'phase': args.phase, 'stack': name, 'stackId': response['StackId'], 'changeSet': response['Id'], 'templateSha256': digest(body), 'parameters': params, 'resources': resources, 'changeResources': change_resources, 'core': state, 'services': services, 'package': package, 'preparedAt': datetime.now(timezone.utc).isoformat()}
        if stack:
            receipt['tagUpdates'] = restored['retained']
        write(receipt_path, receipt)
        print(json.dumps({'preparedChangeSet': response['Id'], 'resources': len(resources), 'executionEnabled': False}))
        return
    require(receipt is not None and stack is not None, 'Prepare a concrete change set first')
    require(stack['StackId'] == receipt['stackId'] and stack['StackStatus'] in {'REVIEW_IN_PROGRESS', 'IMPORT_COMPLETE'}, 'Preserve the started deployment; use status')
    require(state == receipt['core'] and services == receipt['services'], 'Core or preserved service changed; inspect before proceeding')
    require(digest(body) == receipt['templateSha256'], 'Local template changed after preparation')
    change = cf.describe_change_set(ChangeSetName=receipt['changeSet'])
    require(not change.get('NextToken'), 'Inspect all change-set pages before proceeding')
    summary = validate_change(change, receipt)
    stored = cf.get_template(ChangeSetName=receipt['changeSet'], TemplateStage='Original')['TemplateBody']
    require(isinstance(stored, str) and digest(stored.encode()) == receipt['templateSha256'], 'Stored change-set template differs from the reviewed source')
    if args.action == 'review':
        receipt['reviewedChanges'] = summary
        write(receipt_path, receipt)
        print(json.dumps({'stack': name, 'changes': summary, 'parameters': receipt['parameters'], 'executionEnabled': False}, indent=2))
    else:
        require(receipt.get('reviewedChanges') == summary, 'Review this concrete change set before execution')
        cf.execute_change_set(ChangeSetName=receipt['changeSet'])
        receipt['executedAt'] = datetime.now(timezone.utc).isoformat()
        write(receipt_path, receipt)
        print(json.dumps({'executedChangeSet': receipt['changeSet'], 'executionEnabled': False}))


if __name__ == '__main__':
    try:
        main()
    except (ValueError, ClientError) as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
