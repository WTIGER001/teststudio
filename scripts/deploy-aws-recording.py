import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import time

import boto3
from botocore.exceptions import ClientError


ROOT = Path(__file__).resolve().parents[1]


def command(arguments, **options):
    result = subprocess.run(arguments, capture_output=True, text=True, **options)
    if result.returncode:
        raise RuntimeError('Command failed: ' + ' '.join(arguments[:2]))
    return result.stdout.strip()


def write(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2) + '\n')
    path.chmod(0o600)


def identities():
    with tempfile.TemporaryDirectory(prefix='teststudio-recording-identities-') as temporary:
        root = Path(temporary)

        def openssl(*arguments):
            command(['openssl', *arguments], cwd=root)

        openssl('req', '-x509', '-newkey', 'rsa:3072', '-nodes', '-keyout', 'ca.key', '-out', 'ca.crt', '-days', '397', '-subj', '/CN=TestStudio private recording transport', '-addext', 'basicConstraints=critical,CA:TRUE', '-addext', 'keyUsage=critical,keyCertSign,cRLSign')
        certificates = {}
        for name, hostname, usage in [('server', 'worker.teststudio-recording.internal', 'serverAuth'), ('api', 'teststudio-api.recording.internal', 'clientAuth')]:
            openssl('req', '-newkey', 'rsa:3072', '-nodes', '-keyout', name + '.key', '-out', name + '.csr', '-subj', '/CN=' + hostname)
            (root / 'extensions').write_text('basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature,keyEncipherment\nextendedKeyUsage=' + usage + '\nsubjectAltName=DNS:' + hostname + '\n')
            openssl('x509', '-req', '-in', name + '.csr', '-CA', 'ca.crt', '-CAkey', 'ca.key', '-CAcreateserial', '-out', name + '.crt', '-days', '90', '-extfile', 'extensions')
            certificates[name] = {'certificate': (root / (name + '.crt')).read_text(), 'key': (root / (name + '.key')).read_text()}
        return (root / 'ca.crt').read_text(), certificates


def main():
    parser = argparse.ArgumentParser(description='Deploy only the dedicated private TestStudio recording host.')
    parser.add_argument('action', choices=['prepare', 'review', 'execute', 'status', 'install', 'upgrade'])
    parser.add_argument('--profile', required=True)
    parser.add_argument('--region', required=True)
    parser.add_argument('--account', required=True)
    parser.add_argument('--core-stack', required=True)
    parser.add_argument('--stack', required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--runtime-image')
    parser.add_argument('--gateway-image')
    parser.add_argument('--seccomp', type=Path)
    parser.add_argument('--origin-policy', type=Path)
    parser.add_argument('--package-receipt', type=Path)
    args = parser.parse_args()
    os.chdir(ROOT)
    session = boto3.Session(profile_name=args.profile, region_name=args.region)
    assert session.client('sts').get_caller_identity()['Account'] == args.account, 'Unexpected AWS account'
    cf, ssm, secrets = session.client('cloudformation'), session.client('ssm'), session.client('secretsmanager')
    core = cf.describe_stacks(StackName=args.core_stack)['Stacks'][0]
    assert {'Key': 'Application', 'Value': 'TestStudio'} in core.get('Tags', []), 'Core ownership is required'
    assert core['StackStatus'] in ['CREATE_COMPLETE', 'UPDATE_COMPLETE'], 'Preserve the core deployment in progress'
    values = {v['ParameterKey']: v['ParameterValue'] for v in core['Parameters']}
    core_outputs = {v['OutputKey']: v['OutputValue'] for v in core['Outputs']}
    receipt_path = args.output / 'host.json'
    receipt = json.loads(receipt_path.read_text()) if receipt_path.exists() else None
    try:
        stack = cf.describe_stacks(StackName=args.stack)['Stacks'][0]
        if stack['StackStatus'] == 'REVIEW_IN_PROGRESS' and receipt:
            pending = cf.describe_change_set(StackName=args.stack, ChangeSetName=receipt['changeSet'])
            assert pending['StackId'] == stack['StackId'] and {'Key': 'Application', 'Value': 'TestStudio'} in pending.get('Tags', []), 'Recording change-set ownership is required'
        else:
            assert {'Key': 'Application', 'Value': 'TestStudio'} in stack.get('Tags', []), 'Recording stack ownership is required'
    except ClientError as error:
        if 'does not exist' not in str(error):
            raise
        stack = None
    if args.action == 'prepare':
        if receipt:
            assert receipt['account'] == args.account and receipt['stack'] == args.stack
            print(json.dumps({'reusedChangeSet': receipt['changeSet']}))
            return
        assert stack is None, 'Inspect and drain the installed host before preparing a host update'
        image = ssm.get_parameter(Name='/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-arm64')['Parameter']['Value']
        params = {'VpcId': values['VpcId'], 'PrivateSubnet': values['PrivateSubnets'].split(',')[0], 'ApiSecurityGroup': core_outputs['TaskSecurityGroupId'], 'DatabaseSecurityGroup': values['DatabaseSecurityGroupId'], 'DatabaseSecretArn': values['DatabaseSecretArn'], 'ImageId': image}
        response = cf.create_change_set(StackName=args.stack, ChangeSetName='private-recording-host', ChangeSetType='CREATE', TemplateBody=(ROOT / 'deploy/aws/recording.yaml').read_text(), Parameters=[{'ParameterKey': k, 'ParameterValue': v} for k, v in params.items()], Capabilities=['CAPABILITY_IAM'], Tags=[{'Key': 'Application', 'Value': 'TestStudio'}])
        receipt = {'account': args.account, 'stack': args.stack, 'changeSet': response['Id'], 'parameters': params}
        write(receipt_path, receipt)
        print(json.dumps({'preparedChangeSet': response['Id'], 'instanceType': 't4g.medium', 'privateOnly': True}))
        return
    assert receipt and receipt['account'] == args.account and receipt['stack'] == args.stack, 'An owned deployment receipt is required'
    if args.action in ['review', 'execute']:
        change = cf.describe_change_set(StackName=args.stack, ChangeSetName=receipt['changeSet'])
        assert change['Status'] == 'CREATE_COMPLETE', 'Inspect change-set creation before execution'
        summary = [{k: v['ResourceChange'].get(k) for k in ['Action', 'LogicalResourceId', 'ResourceType', 'Replacement']} for v in change.get('Changes', [])]
        expected = {'PackageBucket', 'PackagePolicy', 'BrowserRepository', 'GatewayRepository', 'ServerIdentity', 'ApiIdentity', 'WorkerSecurityGroup', 'DatabaseAccess', 'WorkerRole', 'WorkerProfile', 'PrivateZone', 'Worker', 'PrivateRecord'}
        assert {v['LogicalResourceId'] for v in summary} == expected and all(v['Action'] == 'Add' for v in summary), 'The dedicated-host change set must only add its reviewed resources'
        if args.action == 'review':
            receipt['reviewedChanges'] = summary
            write(receipt_path, receipt)
            print(json.dumps({'changes': summary}, indent=2))
        else:
            assert receipt.get('reviewedChanges') == summary and change['ExecutionStatus'] == 'AVAILABLE', 'Review the concrete available change set first'
            cf.execute_change_set(StackName=args.stack, ChangeSetName=receipt['changeSet'])
            print(json.dumps({'executedChangeSet': receipt['changeSet']}))
        return
    assert stack, 'The recording stack does not exist'
    if args.action == 'status':
        outputs = {v['OutputKey']: v['OutputValue'] for v in stack.get('Outputs', [])}
        receipt['outputs'] = outputs
        write(receipt_path, receipt)
        instances = ssm.describe_instance_information(Filters=[{'Key': 'InstanceIds', 'Values': [outputs['InstanceId']]}])['InstanceInformationList'] if outputs else []
        print(json.dumps({'status': stack['StackStatus'], 'outputs': outputs, 'managedHostOnline': bool(instances and instances[0]['PingStatus'] == 'Online')}))
        return
    assert stack['StackStatus'] == 'CREATE_COMPLETE', 'The recording host must finish creation before installation'
    outputs = {v['OutputKey']: v['OutputValue'] for v in stack['Outputs']}
    package_receipt = receipt_path
    if args.action == 'upgrade':
        assert args.package_receipt and args.package_receipt != receipt_path and args.origin_policy, 'Upgrade requires a separate checkpoint and explicit reviewed origin policy'
        package_receipt = args.package_receipt
        receipt = json.loads(package_receipt.read_text()) if package_receipt.exists() else {'account': args.account, 'stack': args.stack, 'outputs': outputs}
        assert receipt['account'] == args.account and receipt['stack'] == args.stack and receipt['outputs']['InstanceId'] == outputs['InstanceId'], 'Upgrade checkpoint ownership does not match'
    if receipt.get('installCommand'):
        result = ssm.get_command_invocation(CommandId=receipt['installCommand'], InstanceId=outputs['InstanceId'])
        print(json.dumps({'installCommand': receipt['installCommand'], 'status': result['Status'], 'summary': result.get('StandardOutputContent', '')[-2000:]}))
        return
    assert command(['git', 'status', '--porcelain']) == '', 'Commit implementation before building the installation package'
    commit = command(['git', 'rev-parse', 'HEAD'])
    assert args.runtime_image and args.gateway_image and args.seccomp, 'Select the verified immutable runtime, gateway, and sandbox policy'
    if args.origin_policy:
        command(['go', 'run', './cmd/teststudio-recording-worker', '--validate-origin-policy', '--origin-policy=' + str(args.origin_policy.resolve())], env={**os.environ, 'GOWORK': 'off'})
    current = []
    for name in ['ServerSecretArn', 'ApiSecretArn']:
        current.append(bool(secrets.describe_secret(SecretId=outputs[name]).get('VersionIdsToStages')))
    assert current[0] == current[1], 'Inspect incomplete identity provisioning before continuing'
    if not current[0]:
        ca, pair = identities()
        server = {**pair['server'], 'clientCA': ca}
        config = {'origin': outputs['WorkerOrigin'], 'serverCAEnvironment': 'TESTSTUDIO_RECORDING_CA', 'clientCertificateEnvironment': 'TESTSTUDIO_RECORDING_CERTIFICATE', 'clientKeyEnvironment': 'TESTSTUDIO_RECORDING_KEY'}
        api = {'configuration': json.dumps(config), 'serverCA': ca, 'clientCertificate': pair['api']['certificate'], 'clientKey': pair['api']['key']}
        secrets.put_secret_value(SecretId=outputs['ServerSecretArn'], SecretString=json.dumps(server))
        secrets.put_secret_value(SecretId=outputs['ApiSecretArn'], SecretString=json.dumps(api))
        pair = server = api = None
    ecr = session.client('ecr')

    def publish(image, repository):
        tag = 'source-' + image.removeprefix('sha256:')
        name = repository.split('/', 1)[1]
        try:
            details = ecr.describe_images(repositoryName=name, imageIds=[{'imageTag': tag}])['imageDetails'][0]
        except ecr.exceptions.ImageNotFoundException:
            endpoint = command(['docker', 'context', 'inspect', '--format', '{{.Endpoints.docker.Host}}'])
            with tempfile.TemporaryDirectory(prefix='teststudio-recording-registry-') as private:
                docker = ['docker', '--config', private, '--host', endpoint]
                password = command(['aws', '--profile', args.profile, '--region', args.region, 'ecr', 'get-login-password'])
                command(docker + ['login', '--username', 'AWS', '--password-stdin', repository.split('/')[0]], input=password)
                password = None
                command(docker + ['tag', image, repository + ':' + tag])
                command(docker + ['push', repository + ':' + tag])
            details = ecr.describe_images(repositoryName=name, imageIds=[{'imageTag': tag}])['imageDetails'][0]
        return repository + '@' + details['imageDigest']

    runtime = publish(args.runtime_image, outputs['BrowserRepository'])
    gateway = publish(args.gateway_image, outputs['GatewayRepository'])
    with tempfile.TemporaryDirectory(prefix='teststudio-recording-package-') as temporary:
        root = Path(temporary)
        command(['go', 'build', '-trimpath', '-ldflags=-s -w', '-o', str(root / 'teststudio-recording-worker'), './cmd/teststudio-recording-worker'], env={**os.environ, 'CGO_ENABLED': '0', 'GOOS': 'linux', 'GOARCH': 'arm64', 'GOWORK': 'off'})
        for name in ['install.py', 'start.py']:
            (root / name).write_bytes((ROOT / 'deploy/recording' / name).read_bytes())
        (root / 'seccomp.json').write_bytes(args.seccomp.read_bytes())
        (root / 'rds-ca-bundle.pem').write_bytes((ROOT / 'deploy/service/rds-ca-bundle.pem').read_bytes())
        origin_policy = json.loads(args.origin_policy.read_text()) if args.origin_policy else {}
        assert isinstance(origin_policy, dict) and len(origin_policy) <= 64 and all(isinstance(k, str) and isinstance(v, list) and len(v) <= 16 and all(isinstance(origin, str) for origin in v) for k, v in origin_policy.items()), 'Origin policy requires a bounded map of reviewed exact website origins'
        manifest = {'sourceCommit': commit, 'region': args.region, 'capacity': 1, 'runtimeImage': runtime, 'gatewayImage': gateway, 'serverSecret': outputs['ServerSecretArn'], 'databaseSecret': values['DatabaseSecretArn'], 'originPolicy': origin_policy}
        write(root / 'manifest.json', manifest)
        package = root / 'package.tar.gz'
        with tarfile.open(package, 'w:gz') as archive:
            for path in sorted(root.iterdir()):
                if path != package:
                    archive.add(path, arcname=path.name)
        sha = hashlib.sha256(package.read_bytes()).hexdigest()
        key = 'packages/' + sha + '.tar.gz'
        session.client('s3').upload_file(str(package), outputs['PackageBucket'], key, ExtraArgs={'ServerSideEncryption': 'AES256'})
    drain = 'systemctl stop teststudio-recording.service\n' if args.action == 'upgrade' else ''
    script = 'set -eu\ncloud-init status --wait >/dev/null\nprivate=$(mktemp -d /var/tmp/teststudio-recording.XXXXXX)\ntrap \'rm -rf "$private"\' EXIT\naws s3 cp s3://' + outputs['PackageBucket'] + '/' + key + ' "$private/package.tar.gz" --region ' + args.region + ' --only-show-errors\necho "' + sha + '  $private/package.tar.gz" | sha256sum --check --status\ntar -xzf "$private/package.tar.gz" -C "$private"\n' + drain + 'python3 "$private/install.py"\nsystemctl is-active teststudio-recording.service\n'
    result = ssm.send_command(InstanceIds=[outputs['InstanceId']], DocumentName='AWS-RunShellScript', Parameters={'commands': [script], 'executionTimeout': ['1200']}, TimeoutSeconds=1200, Comment='Install committed private TestStudio recording worker')
    receipt.update(outputs=outputs, packageSHA256=sha, packageKey=key, runtimeImage=runtime, gatewayImage=gateway, sourceCommit=commit, installCommand=result['Command']['CommandId'])
    write(package_receipt, receipt)
    print(json.dumps({'installCommand': receipt['installCommand'], 'sourceCommit': commit, 'packageSHA256': sha}))


if __name__ == '__main__':
    try:
        main()
    except (AssertionError, RuntimeError, ClientError) as error:
        if isinstance(error, ClientError):
            raise SystemExit('AWS recording deployment request failed: ' + error.response['Error']['Code']) from None
        raise SystemExit(str(error)) from None
