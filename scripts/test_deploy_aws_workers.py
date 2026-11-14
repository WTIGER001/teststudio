import copy
import importlib.util
from pathlib import Path
import unittest
from unittest.mock import Mock


spec = importlib.util.spec_from_file_location('deploy_aws_workers', Path(__file__).with_name('deploy-aws-workers.py'))
deployment = importlib.util.module_from_spec(spec)
spec.loader.exec_module(deployment)
restore_spec = importlib.util.spec_from_file_location('restore_aws_workers', Path(__file__).with_name('restore-aws-workers.py'))
restoration = importlib.util.module_from_spec(restore_spec)
restore_spec.loader.exec_module(restoration)


class WorkerChangeReviewTests(unittest.TestCase):
    def setUp(self):
        self.receipt = {'phase': 'workers', 'stack': 'teststudioapi-workers', 'stackId': 'owned-stack', 'changeSet': 'owned-change', 'parameters': {'Stage': 'prepare', 'EnablePreparation': 'false', 'EnableNewBuilds': 'false'}, 'resources': {'ControllerPolicy': 'AWS::IAM::ManagedPolicy', 'WorkerVpc': 'AWS::EC2::VPC'}}
        self.change = {'Status': 'CREATE_COMPLETE', 'ExecutionStatus': 'AVAILABLE', 'StackName': self.receipt['stack'], 'StackId': 'owned-stack', 'ChangeSetId': 'owned-change', 'Parameters': [{'ParameterKey': key, 'ParameterValue': value} for key, value in self.receipt['parameters'].items()], 'Tags': [deployment.OWNERSHIP], 'Changes': [{'ResourceChange': {'Action': 'Add', 'LogicalResourceId': key, 'ResourceType': value}} for key, value in self.receipt['resources'].items()]}

    def test_review_accepts_only_the_frozen_initial_additions(self):
        result = deployment.validate_change(self.change, self.receipt)
        self.assertEqual([item['LogicalResourceId'] for item in result], ['ControllerPolicy', 'WorkerVpc'])

    def test_restore_imports_and_followup_additions_have_distinct_review(self):
        imported = copy.deepcopy(self.change)
        for item in imported['Changes']:
            item['ResourceChange']['Action'] = 'Import'
        with self.assertRaises(ValueError):
            deployment.validate_change(imported, self.receipt)
        imported['Tags'] = []
        self.assertEqual(len(deployment.validate_change(imported, {**self.receipt, 'phase': 'restore'})), 2)
        followup = copy.deepcopy(self.change)
        followup['Changes'].pop()
        receipt = {**self.receipt, 'changeResources': {'ControllerPolicy': 'AWS::IAM::ManagedPolicy'}}
        self.assertEqual(len(deployment.validate_change(followup, receipt)), 1)
        with self.assertRaises(ValueError):
            deployment.validate_change(self.change, receipt)

    def test_review_rejects_activated_or_changed_parameters(self):
        for key, value in [('Stage', 'ready'), ('EnablePreparation', 'true'), ('EnableNewBuilds', 'true')]:
            with self.subTest(key=key):
                changed = copy.deepcopy(self.change)
                next(item for item in changed['Parameters'] if item['ParameterKey'] == key)['ParameterValue'] = value
                with self.assertRaisesRegex(ValueError, 'parameters changed'):
                    deployment.validate_change(changed, self.receipt)

    def test_imported_tags_cannot_expand_into_data_changes(self):
        tagged = copy.deepcopy(self.change)
        resource = {'Action': 'Modify', 'LogicalResourceId': 'Journals', 'ResourceType': 'AWS::EFS::FileSystem', 'PhysicalResourceId': 'owned-filesystem', 'Replacement': 'False', 'Scope': ['Tags'], 'Details': [{'Target': {'Attribute': 'Tags', 'RequiresRecreation': 'Never'}}]}
        tagged['Changes'].append({'ResourceChange': resource})
        receipt = {**self.receipt, 'tagUpdates': {'Journals': 'owned-filesystem'}}
        self.assertEqual(len(deployment.validate_change(tagged, receipt)), 3)
        for key, value in [('Scope', ['Properties']), ('Replacement', 'True'), ('PhysicalResourceId', 'foreign-filesystem'), ('Details', [{'Target': {'Attribute': 'Properties', 'RequiresRecreation': 'Never'}}])]:
            with self.subTest(key=key):
                changed = copy.deepcopy(tagged)
                changed['Changes'][-1]['ResourceChange'][key] = value
                with self.assertRaises(ValueError):
                    deployment.validate_change(changed, receipt)

    def test_review_rejects_resource_expansion_removal_or_replacement(self):
        variants = []
        for action in ['Modify', 'Remove']:
            changed = copy.deepcopy(self.change)
            changed['Changes'][0]['ResourceChange']['Action'] = action
            variants.append(changed)
        changed = copy.deepcopy(self.change)
        changed['Changes'].append({'ResourceChange': {'Action': 'Add', 'LogicalResourceId': 'Unreviewed', 'ResourceType': 'AWS::IAM::Role'}})
        variants.append(changed)
        changed = copy.deepcopy(self.change)
        changed['Changes'][0]['ResourceChange']['Replacement'] = 'True'
        variants.append(changed)
        changed = copy.deepcopy(self.change)
        changed['Changes'].pop()
        variants.append(changed)
        for changed in variants:
            with self.subTest(changes=changed['Changes']):
                with self.assertRaises(ValueError):
                    deployment.validate_change(changed, self.receipt)

    def test_review_rejects_foreign_identity_and_pending_changes(self):
        for key, value in [('StackId', 'foreign-stack'), ('StackName', 'foreign-stack'), ('ChangeSetId', 'foreign-change'), ('Tags', []), ('Status', 'CREATE_IN_PROGRESS'), ('ExecutionStatus', 'EXECUTE_IN_PROGRESS')]:
            with self.subTest(key=key):
                with self.assertRaises(ValueError):
                    deployment.validate_change({**self.change, key: value}, self.receipt)

    def test_bootstrap_contains_only_private_storage_and_certificate(self):
        _, template = deployment.template_for('bootstrap')
        resources = deployment.expected_resources(template, {'ExchangeHostname': 'teststudioapi-workers.example.test', 'HostedZoneId': 'Z1234'}, 'us-west-2')
        self.assertEqual(resources, {'PackageBucket': 'AWS::S3::Bucket', 'ExchangeCertificate': 'AWS::CertificateManager::Certificate'})
        bucket = template['Resources']['PackageBucket']
        self.assertEqual(bucket['DeletionPolicy'], 'Retain')
        self.assertTrue(all(bucket['Properties']['PublicAccessBlockConfiguration'].values()))

    def test_service_inventory_does_not_require_another_application(self):
        ecs = Mock()
        ecs.get_paginator.return_value.paginate.return_value = [{'serviceArns': ['arn:owned/studio']}]
        ecs.describe_services.return_value = {'services': [{'serviceName': 'studio', 'status': 'ACTIVE', 'desiredCount': 1, 'runningCount': 1, 'pendingCount': 0, 'deployments': [{'id': 'owned-deployment', 'rolloutState': 'COMPLETED'}]}]}
        state = deployment.service_state(ecs, {'outputs': {'ClusterArn': 'owned-cluster', 'ServiceName': 'studio'}})
        self.assertEqual(state, {'studio': {'deployment': 'owned-deployment', 'desired': 1}})

    def test_recovery_preserves_only_exact_retained_inventory_after_rollback(self):
        cf = Mock()
        cf.describe_stacks.return_value = {'Stacks': [{'StackId': 'failed-stack', 'StackStatus': 'ROLLBACK_COMPLETE', 'Tags': [deployment.OWNERSHIP]}]}
        cf.get_template.return_value = {'TemplateBody': 'reviewed-template'}
        failed = {'stack': 'teststudioapi-workers', 'stackId': 'failed-stack', 'templateSha256': deployment.digest(b'reviewed-template'), 'resources': {key: 'resource-type' for key in restoration.RETAINED}}
        resources = [{'LogicalResourceId': key, 'ResourceStatus': 'DELETE_SKIPPED', 'ResourceType': 'resource-type', 'PhysicalResourceId': key + '-owned'} for key in restoration.RETAINED]
        cf.describe_stack_resources.return_value = {'StackResources': resources}
        self.assertEqual(set(restoration.retained_resources(cf, failed)), set(restoration.RETAINED))
        cf.describe_stacks.return_value['Stacks'][0]['StackStatus'] = 'ROLLBACK_IN_PROGRESS'
        with self.assertRaises(ValueError):
            restoration.retained_resources(cf, failed)
        cf.describe_stacks.return_value['Stacks'][0]['StackStatus'] = 'ROLLBACK_COMPLETE'
        resources[0]['ResourceStatus'] = 'CREATE_COMPLETE'
        with self.assertRaises(ValueError):
            restoration.retained_resources(cf, failed)
        resources[0]['ResourceStatus'] = 'DELETE_SKIPPED'
        cf.get_template.return_value['TemplateBody'] = 'changed-retain-policies'
        with self.assertRaises(ValueError):
            restoration.retained_resources(cf, failed)
        cf.delete_stack.assert_not_called()


if __name__ == '__main__':
    unittest.main()
