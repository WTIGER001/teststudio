import argparse
import importlib.util
import json
from pathlib import Path

import boto3


spec = importlib.util.spec_from_file_location('deploy_aws_workers', Path(__file__).with_name('deploy-aws-workers.py'))
deployment = importlib.util.module_from_spec(spec)
spec.loader.exec_module(deployment)
RETAINED = {'GoRepository': 'RepositoryName', 'PlaywrightRepository': 'RepositoryName', 'Journals': 'FileSystemId', 'JournalAccessPoint': 'AccessPointId', 'PreparationLog': 'LogGroupName'}


def retained_resources(cf, failed):
    stack = deployment.find_stack(cf, failed['stack'])
    deployment.require(stack and stack['StackId'] == failed['stackId'] and stack['StackStatus'] == 'ROLLBACK_COMPLETE' and deployment.OWNERSHIP in stack.get('Tags', []), 'Wait for the owned initial rollback to finish')
    original = cf.get_template(StackName=stack['StackId'], TemplateStage='Original')['TemplateBody']
    deployment.require(isinstance(original, str) and deployment.digest(original.encode()) == failed['templateSha256'], 'Preserve the reviewed retain policies before retiring the failed stack record')
    resources = cf.describe_stack_resources(StackName=stack['StackId'])['StackResources']
    remaining = {item['LogicalResourceId']: item for item in resources if item['ResourceStatus'] != 'DELETE_COMPLETE'}
    deployment.require(set(remaining) == set(RETAINED) and all(item['ResourceStatus'] == 'DELETE_SKIPPED' and item['ResourceType'] == failed['resources'][key] for key, item in remaining.items()), 'Inspect unexpected rollback resources before recovery')
    return remaining


def main():
    parser = argparse.ArgumentParser(description='Preserve and import retained resources after an initial TestStudio worker rollback.')
    parser.add_argument('action', choices=['capture', 'retire', 'prepare', 'review', 'execute', 'status'])
    for name in ['profile', 'region', 'account', 'core-stack']:
        parser.add_argument('--' + name, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    session = boto3.Session(profile_name=args.profile, region_name=args.region)
    deployment.require(session.client('sts').get_caller_identity()['Account'] == args.account, 'Unexpected AWS account')
    cf = session.client('cloudformation')
    path = args.output / 'restore.json'
    archive = args.output / 'workers-failed.json'
    failed_path = archive if archive.exists() else args.output / 'workers.json'
    failed = json.loads(failed_path.read_text())
    deployment.require(failed['stack'] == args.core_stack + '-workers' and failed['account'] == args.account and failed['region'] == args.region, 'The failure receipt belongs to another installation')
    deployment.require(failed['parameters']['Stage'] == 'prepare' and all(failed['parameters'][key] == 'false' for key in ['EnablePreparation', 'EnableNewBuilds', 'EnableEnvironment']), 'Recovery is limited to disabled initial infrastructure')
    core = deployment.find_stack(cf, args.core_stack)
    deployment.require(core and core['StackStatus'] in deployment.COMPLETE and deployment.core_state(core) == failed['core'], 'Core inputs changed')
    deployment.require(deployment.service_state(session.client('ecs'), failed['core']) == failed['services'], 'Preserve changed service deployments')
    receipt = json.loads(path.read_text()) if path.exists() else None
    if args.action == 'capture':
        deployment.require(receipt is None, 'Preserve the existing restoration receipt')
        remaining = retained_resources(cf, failed)
        receipt = {'account': args.account, 'region': args.region, 'phase': 'restore', 'stack': failed['stack'], 'failedStackId': failed['stackId'], 'parameters': failed['parameters'], 'core': failed['core'], 'services': failed['services'], 'resources': {key: item['ResourceType'] for key, item in remaining.items()}, 'retained': {key: item['PhysicalResourceId'] for key, item in remaining.items()}}
        deployment.write(path, receipt)
        print(json.dumps({'reviewedRetainedResources': receipt['retained'], 'nextAction': 'retire only the failed stack record; preserve all five resources'}, indent=2))
        return
    deployment.require(receipt is not None, 'Capture and inspect the completed rollback first')
    if args.action == 'retire':
        remaining = retained_resources(cf, failed)
        deployment.require({key: item['PhysicalResourceId'] for key, item in remaining.items()} == receipt['retained'], 'Retained resources changed')
        cf.delete_stack(StackName=failed['stackId'])
        receipt['retiredStackRecord'] = True
        deployment.write(path, receipt)
        print(json.dumps({'deletedFailedStackRecord': failed['stackId'], 'retainedResourcesDeleted': False}))
        return
    if args.action == 'prepare':
        if receipt.get('changeSet'):
            print(json.dumps({'reusedChangeSet': receipt['changeSet']}))
            return
        deployment.require(receipt.get('retiredStackRecord') and deployment.find_stack(cf, receipt['stack']) is None, 'Wait for deletion of the failed stack record')
        _, full = deployment.template_for('workers')
        template = {'AWSTemplateFormatVersion': '2010-09-09', 'Parameters': full['Parameters'], 'Resources': {key: full['Resources'][key] for key in RETAINED}}
        body = json.dumps(template, separators=(',', ':'))
        receipt['template'] = template
        request = {'StackName': receipt['stack'], 'ChangeSetName': 'restore-retained-workers', 'ChangeSetType': 'IMPORT', 'TemplateBody': body, 'Parameters': [{'ParameterKey': key, 'ParameterValue': value} for key, value in receipt['parameters'].items()], 'ResourcesToImport': [{'ResourceType': receipt['resources'][key], 'LogicalResourceId': key, 'ResourceIdentifier': {RETAINED[key]: value}} for key, value in receipt['retained'].items()]}
        response = cf.create_change_set(**request)
        receipt.update({'changeSet': response['Id'], 'stackId': response['StackId']})
        deployment.write(path, receipt)
        print(json.dumps({'preparedImportChangeSet': response['Id'], 'resources': 5, 'retainedResourcesDeleted': False}))
        return
    stack = deployment.find_stack(cf, receipt['stack'])
    deployment.require(stack and stack['StackId'] == receipt['stackId'], 'Inspect the restoration stack')
    if args.action == 'status':
        if stack['StackStatus'] == 'IMPORT_COMPLETE':
            actual = cf.describe_stack_resources(StackName=stack['StackId'])['StackResources']
            deployment.require({item['LogicalResourceId']: item['PhysicalResourceId'] for item in actual} == receipt['retained'], 'Imported resources differ from the retained inventory')
            receipt['importComplete'] = True
            deployment.write(path, receipt)
            if not archive.exists():
                failed_path.rename(archive)
        print(json.dumps({'status': stack['StackStatus'], 'importComplete': receipt.get('importComplete', False)}))
        return
    change = cf.describe_change_set(ChangeSetName=receipt['changeSet'])
    deployment.require(not change.get('NextToken'), 'Inspect all import changes')
    summary = deployment.validate_change(change, receipt)
    stored = cf.get_template(ChangeSetName=receipt['changeSet'], TemplateStage='Original')['TemplateBody']
    deployment.require((json.loads(stored) if isinstance(stored, str) else stored) == receipt['template'], 'Import template changed')
    if args.action == 'review':
        receipt['reviewedChanges'] = summary
        deployment.write(path, receipt)
        print(json.dumps({'changes': summary, 'retainedResourcesDeleted': False}, indent=2))
    else:
        deployment.require(receipt.get('reviewedChanges') == summary, 'Review the concrete import change set first')
        cf.execute_change_set(ChangeSetName=receipt['changeSet'])
        print(json.dumps({'executedImportChangeSet': receipt['changeSet']}))


if __name__ == '__main__':
    main()
