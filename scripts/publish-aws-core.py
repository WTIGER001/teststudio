import argparse
import json
import re
from pathlib import Path
import subprocess
import tempfile
import time
import urllib.request
import urllib.parse


def command(args, **kwargs):
    result = subprocess.run(args, capture_output=True, text=True, **kwargs)
    if result.returncode:
        raise RuntimeError('Command failed: ' + ' '.join(args[:2]))
    return result.stdout


def main():
    parser = argparse.ArgumentParser(description='Prepare, review, execute, and finish a TestStudio-owned core application release.')
    parser.add_argument('action', choices=['prepare', 'review', 'execute', 'finish'])
    parser.add_argument('--profile', required=True)
    parser.add_argument('--region', required=True)
    parser.add_argument('--account', required=True)
    parser.add_argument('--stack', required=True)
    parser.add_argument('--receipt', required=True)
    parser.add_argument('--image-receipt')
    parser.add_argument('--public-status', action='store_true', help='Include the scoped public status framing policy and CloudFront behavior.')
    parser.add_argument('--status-logos', action='store_true', help='Include HTTPS and uploaded logo image sources in the UI and public status response policies.')
    parser.add_argument('--recording-worker-secret-arn', help='Select the dedicated private recording transport identity and disable desktop pairing.')
    parser.add_argument('--local-accounts', choices=['true', 'false'], help='Explicitly select hosted password sign-in; omission preserves the installed setting.')
    parser.add_argument('--github-source-secret-arn', help='Select an owned GitHub API credential secret for one project.')
    parser.add_argument('--github-source-project')
    parser.add_argument('--github-source-reference', default='github-source')
    args = parser.parse_args()
    assert bool(args.github_source_secret_arn) == bool(args.github_source_project), 'Supply both GitHub source secret and project'
    if args.github_source_secret_arn:
        assert all(re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._-]{0,127}', value) for value in [args.github_source_project, args.github_source_reference]), 'GitHub source scope must use valid identifiers'
    aws_base = ['aws', '--profile', args.profile, '--region', args.region, '--no-cli-pager']

    def aws(*items):
        output = command(aws_base + list(items))
        return json.loads(output) if output.strip() else {}

    assert aws('sts', 'get-caller-identity')['Account'] == args.account, 'Unexpected AWS account'
    stack = aws('cloudformation', 'describe-stacks', '--stack-name', args.stack)['Stacks'][0]
    assert {'Key': 'Application', 'Value': 'TestStudio'} in stack.get('Tags', []), 'TestStudio stack ownership is required'
    outputs = {v['OutputKey']: v['OutputValue'] for v in stack['Outputs']}
    params = {v['ParameterKey']: v['ParameterValue'] for v in stack['Parameters']}
    assert outputs['ServiceName'] == args.stack, 'Unexpected owned service name'
    receipt_path = Path(args.receipt)
    receipt_path.parent.mkdir(parents=True, exist_ok=True)
    if args.action == 'prepare':
        assert stack['StackStatus'] in ['CREATE_COMPLETE', 'UPDATE_COMPLETE'], 'Inspect the existing deployment before starting another'
        build = json.loads(Path(args.image_receipt).read_text())
        assert not build['buildInfo']['modified'], 'Build the release from committed implementation source'
        assert build['buildInfo']['commit'] == command(['git', 'rev-parse', 'HEAD']).strip(), 'The artifact must identify the current committed source'
        if receipt_path.exists():
            receipt = json.loads(receipt_path.read_text())
            assert receipt['image'] == build['image'], 'Use a new receipt for a different release'
            assert args.local_accounts is None or receipt.get('localAccounts') == args.local_accounts, 'Use a new receipt for different local sign-in settings'
            assert not args.github_source_secret_arn or receipt.get('githubSource') == {'secretArn': args.github_source_secret_arn, 'project': args.github_source_project, 'reference': args.github_source_reference}, 'Use a new receipt for different GitHub source settings'
            print(json.dumps({'reusedChangeSet': receipt['changeSet'], 'buildId': build['buildInfo']['buildId']}))
            return
        registry = args.account + '.dkr.ecr.' + args.region + '.amazonaws.com'
        repository = params['ServiceRepositoryName']
        assert repository.startswith(args.stack + '/'), 'Unexpected registry ownership'
        tag = 'build-' + build['buildInfo']['buildId']
        with tempfile.TemporaryDirectory(prefix='teststudio-release-docker-') as private:
            endpoint = command(['docker', 'context', 'inspect', '--format', '{{.Endpoints.docker.Host}}']).strip()
            docker = ['docker', '--config', private, '--host', endpoint]
            password = command(aws_base + ['ecr', 'get-login-password'])
            command(docker + ['login', '--username', 'AWS', '--password-stdin', registry], input=password)
            password = None
            command(docker + ['tag', build['image'], registry + '/' + repository + ':' + tag])
            command(docker + ['push', registry + '/' + repository + ':' + tag])
        details = aws('ecr', 'describe-images', '--repository-name', repository, '--image-ids', 'imageTag=' + tag)['imageDetails'][0]
        parameters = [{'ParameterKey': v['ParameterKey'], 'ParameterValue': details['imageDigest']} if v['ParameterKey'] == 'ServiceImageDigest' else {'ParameterKey': v['ParameterKey'], 'UsePreviousValue': True} for v in stack['Parameters']]
        if args.local_accounts is not None:
            assert 'LocalAccounts' in params, 'The selected installation must support managed local accounts'
            parameters = [v for v in parameters if v['ParameterKey'] != 'LocalAccounts']
            parameters.append({'ParameterKey': 'LocalAccounts', 'ParameterValue': args.local_accounts})
        if args.recording_worker_secret_arn:
            assert args.recording_worker_secret_arn.startswith('arn:aws:secretsmanager:' + args.region + ':' + args.account + ':secret:'), 'The recording transport secret must belong to the selected installation account'
            secret = aws('secretsmanager', 'describe-secret', '--secret-id', args.recording_worker_secret_arn)
            assert any(v == {'Key': 'Application', 'Value': 'TestStudio'} for v in secret.get('Tags', [])), 'The recording identity must be TestStudio-owned'
            parameters = [v for v in parameters if v['ParameterKey'] not in ['RecordingCompanion', 'RecordingWorkerSecretArn']]
            parameters.extend([{'ParameterKey': 'RecordingCompanion', 'ParameterValue': 'false'}, {'ParameterKey': 'RecordingWorkerSecretArn', 'ParameterValue': args.recording_worker_secret_arn}])
        github_source = None
        if args.github_source_secret_arn:
            assert args.github_source_secret_arn.startswith('arn:aws:secretsmanager:' + args.region + ':' + args.account + ':secret:'), 'GitHub source secret must belong to the selected account and region'
            secret = aws('secretsmanager', 'describe-secret', '--secret-id', args.github_source_secret_arn)
            assert {'Key': 'Application', 'Value': 'TestStudio'} in secret.get('Tags', []), 'GitHub source secret must be TestStudio-owned'
            assert {'Key': 'Project', 'Value': args.github_source_project} in secret.get('Tags', []), 'GitHub source secret must belong to the selected project'
            github_source = {'secretArn': args.github_source_secret_arn, 'project': args.github_source_project, 'reference': args.github_source_reference}
            selected = {'GitHubSourceSecretArn': github_source['secretArn'], 'GitHubSourceProject': github_source['project'], 'GitHubSourceReference': github_source['reference']}
            parameters = [v for v in parameters if v['ParameterKey'] not in selected]
            parameters.extend({'ParameterKey': key, 'ParameterValue': value} for key, value in selected.items())
        with tempfile.TemporaryDirectory(prefix='teststudio-release-parameters-') as directory:
            path = Path(directory) / 'parameters.json'
            path.write_text(json.dumps(parameters)); path.chmod(0o600)
            template = ['--template-body', 'file://deploy/aws/service.yaml'] if args.public_status or args.status_logos or args.recording_worker_secret_arn or github_source else ['--use-previous-template']
            result = aws('cloudformation', 'create-change-set', '--stack-name', args.stack, '--change-set-name', 'application-' + tag, '--change-set-type', 'UPDATE', *template, '--parameters', 'file://' + str(path), '--capabilities', 'CAPABILITY_IAM')
        receipt = {**build, 'account': args.account, 'stack': args.stack, 'changeSet': result['Id'], 'registryDigest': details['imageDigest'], 'publicStatusPackage': args.public_status or args.status_logos, 'statusLogosPackage': args.status_logos, 'recordingWorkerPackage': bool(args.recording_worker_secret_arn), 'githubSource': github_source, 'localAccounts': args.local_accounts, 'preparedAt': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()), 'outputs': outputs}
        receipt_path.write_text(json.dumps(receipt, indent=2) + '\n')
        print(json.dumps({'changeSet': receipt['changeSet'], 'digest': receipt['registryDigest'], 'buildId': build['buildInfo']['buildId'], 'privateDockerConfigurationRemoved': True}))
        return
    receipt = json.loads(receipt_path.read_text())
    assert receipt['account'] == args.account and receipt['stack'] == args.stack, 'Receipt ownership does not match'
    if args.action in ['review', 'execute']:
        change = aws('cloudformation', 'describe-change-set', '--stack-name', args.stack, '--change-set-name', receipt['changeSet'])
        if receipt.get('localAccounts') is not None:
            selected = next(v for v in change['Parameters'] if v['ParameterKey'] == 'LocalAccounts')
            assert selected.get('ParameterValue') == receipt['localAccounts'], 'The concrete local sign-in selection differs from the release receipt'
        if receipt.get('githubSource'):
            concrete = {v['ParameterKey']: v.get('ParameterValue') for v in change['Parameters']}
            expected = {'GitHubSourceSecretArn': receipt['githubSource']['secretArn'], 'GitHubSourceProject': receipt['githubSource']['project'], 'GitHubSourceReference': receipt['githubSource']['reference']}
            assert all(concrete.get(key) == value for key, value in expected.items()), 'Concrete GitHub source scope differs from the reviewed selection'
        summary = [{'action': v['ResourceChange']['Action'], 'resource': v['ResourceChange']['LogicalResourceId'], 'type': v['ResourceChange']['ResourceType'], 'replacement': v['ResourceChange'].get('Replacement')} for v in change.get('Changes', [])]
        assert change['Status'] == 'CREATE_COMPLETE', 'The change set is not ready'
        allowed = {'Task': 'AWS::ECS::TaskDefinition', 'Service': 'AWS::ECS::Service'}
        if receipt.get('recordingWorkerPackage') or receipt.get('githubSource'):
            allowed['ExecutionRole'] = 'AWS::IAM::Role'
        if receipt.get('statusLogosPackage'):
            allowed['UiHeaders'] = 'AWS::CloudFront::ResponseHeadersPolicy'
        if receipt.get('publicStatusPackage'):
            allowed['UiDistribution'] = 'AWS::CloudFront::Distribution'
            allowed['PublicStatusHeaders'] = 'AWS::CloudFront::ResponseHeadersPolicy'
            allowed['UiRecord'] = 'AWS::Route53::RecordSet'
            alias = next((entry['ResourceChange'] for entry in change.get('Changes', []) if entry['ResourceChange']['LogicalResourceId'] == 'UiRecord'), None)
            if alias:
                assert alias['Action'] == 'Modify' and alias.get('Replacement') == 'False' and alias.get('PhysicalResourceId') == urllib.parse.urlsplit(outputs['UiUrl']).hostname, 'Preserve the existing UI DNS alias'
                assert alias.get('Details') and all(detail.get('ChangeSource') == 'ResourceAttribute' and detail.get('CausingEntity') == 'UiDistribution.DomainName' and detail.get('Target', {}).get('Name') == 'AliasTarget' and detail['Target'].get('RequiresRecreation') == 'Never' for detail in alias['Details']), 'Only the existing distribution alias dependency can refresh'
        assert summary and all(allowed.get(v['resource']) == v['type'] and (v['action'] == 'Modify' or receipt.get('publicStatusPackage') and v['resource'] == 'PublicStatusHeaders' and v['action'] == 'Add') for v in summary), 'Review unexpected resources separately'
        if args.action == 'review':
            receipt['reviewedChanges'] = summary
            receipt_path.write_text(json.dumps(receipt, indent=2) + '\n')
            print(json.dumps({'status': change['Status'], 'execution': change['ExecutionStatus'], 'changes': summary, 'localAccounts': receipt.get('localAccounts'), 'githubSource': receipt.get('githubSource')}, indent=2))
            return
        assert receipt.get('reviewedChanges') == summary, 'Run review and inspect the concrete changes before execution'
        assert change['ExecutionStatus'] == 'AVAILABLE', 'Preserve an existing or completed deployment'
        assert stack['StackStatus'] in ['CREATE_COMPLETE', 'UPDATE_COMPLETE'], 'A deployment is already in progress'
        aws('cloudformation', 'execute-change-set', '--stack-name', args.stack, '--change-set-name', receipt['changeSet'])
        print(json.dumps({'executedChangeSet': receipt['changeSet']}))
        return
    services = aws('ecs', 'describe-services', '--cluster', outputs['ClusterName'], '--services', outputs['ServiceName'])['services']
    assert len(services) == 1
    service = services[0]
    primary = next(v for v in service['deployments'] if v['status'] == 'PRIMARY')
    if stack['StackStatus'] != 'UPDATE_COMPLETE' or len(service['deployments']) != 1 or primary['rolloutState'] != 'COMPLETED' or service['runningCount'] != service['desiredCount']:
        print(json.dumps({'pending': True, 'stack': stack['StackStatus'], 'rollout': primary['rolloutState']}))
        return
    task_ids = aws('ecs', 'list-tasks', '--cluster', outputs['ClusterName'], '--service-name', outputs['ServiceName'])['taskArns']
    tasks = aws('ecs', 'describe-tasks', '--cluster', outputs['ClusterName'], '--tasks', *task_ids)['tasks']
    assert tasks and all(v['healthStatus'] == 'HEALTHY' and all(c.get('imageDigest') == receipt['registryDigest'] for c in v['containers']) for v in tasks), 'The selected service artifact is not healthy'
    with urllib.request.urlopen(outputs['UiUrl'] + '/api/v1/version', timeout=30) as response:
        assert json.load(response) == receipt['buildInfo'], 'API build identity differs from the selected release'
    ui = Path('web/dist/web/browser')
    assert ui.is_dir()
    build_info = json.loads(Path('.data/build-info.json').read_text())['info']
    assert build_info == receipt['buildInfo'], 'UI build identity differs from the selected release'
    if not receipt.get('uiPublishedAt'):
        command(aws_base + ['s3', 'sync', str(ui), 's3://' + outputs['UiBucketName'], '--exclude', 'index.html', '--cache-control', 'public,max-age=31536000,immutable', '--only-show-errors'])
        command(aws_base + ['s3', 'cp', str(ui / 'index.html'), 's3://' + outputs['UiBucketName'] + '/index.html', '--content-type', 'text/html', '--cache-control', 'no-cache', '--only-show-errors'])
        invalidation = aws('cloudfront', 'create-invalidation', '--distribution-id', outputs['UiDistributionId'], '--paths', '/index.html', '/', '/projects*', '/status/*')['Invalidation']['Id']
        receipt['uiPublishedAt'] = time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()); receipt['invalidation'] = invalidation
        receipt_path.write_text(json.dumps(receipt, indent=2) + '\n')
    print(json.dumps({'complete': True, 'buildId': receipt['buildInfo']['buildId'], 'uiPublishedAt': receipt['uiPublishedAt'], 'invalidation': receipt['invalidation']}))


if __name__ == '__main__':
    try:
        main()
    except (RuntimeError, AssertionError, urllib.error.URLError) as error:
        raise SystemExit(str(error)) from None
