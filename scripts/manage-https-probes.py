import argparse
import importlib.util
import json
from pathlib import Path
import subprocess
import urllib.parse
import urllib.request


def main():
    parser = argparse.ArgumentParser(description='Inspect and configure explicitly selected native HTTPS probes through an owned hosted installation.')
    parser.add_argument('action', choices=['inspect', 'configure', 'run', 'enable', 'check'])
    for name in ['profile', 'region', 'account', 'stack', 'credential-secret']:
        parser.add_argument('--' + name, required=True)
    parser.add_argument('--project')
    parser.add_argument('--config')
    parser.add_argument('--receipt')
    args = parser.parse_args()
    base = ['aws', '--profile', args.profile, '--region', args.region, '--no-cli-pager']

    def aws(*items):
        result = subprocess.run(base + list(items), capture_output=True, text=True)
        if result.returncode:
            raise RuntimeError('AWS ' + ' '.join(items[:2]) + ' failed')
        return json.loads(result.stdout)

    assert aws('sts', 'get-caller-identity')['Account'] == args.account, 'Unexpected AWS account'
    stack = aws('cloudformation', 'describe-stacks', '--stack-name', args.stack)['Stacks'][0]
    assert {'Key': 'Application', 'Value': 'TestStudio'} in stack.get('Tags', []), 'TestStudio installation ownership is required'
    outputs = {v['OutputKey']: v['OutputValue'] for v in stack['Outputs']}
    parameters = {v['ParameterKey']: v['ParameterValue'] for v in stack['Parameters']}
    pool = parameters['BootstrapIssuer'].rsplit('/', 1)[1]
    identity = aws('cognito-idp', 'describe-user-pool', '--user-pool-id', pool)['UserPool']
    module = importlib.util.spec_from_file_location('teststudio_core_api', Path(__file__).with_name('verify-aws-core.py'))
    core = importlib.util.module_from_spec(module); module.loader.exec_module(core)
    api = core.CoreAPI(outputs['UiUrl'], identity['Domain'] + '.auth.' + args.region + '.amazoncognito.com')
    try:
        credential = json.loads(aws('secretsmanager', 'get-secret-value', '--secret-id', args.credential_secret)['SecretString'])
        api.login(credential); credential = None
        if args.action == 'inspect':
            projects = api.call('GET', '/api/v1/projects')['items']
            result = {'projects': [{'id': v['id'], 'name': v['data']['name']} for v in projects]}
            if args.project:
                path = '/api/v1/projects/' + urllib.parse.quote(args.project, safe='')
                result['applications'] = [{'id': v['id'], 'name': v['data']['name']} for v in api.call('GET', path + '/applications')['items']]
                result['suites'] = [{'id': v['id'], 'name': v['data']['name'], 'applications': v['data'].get('applications', [])} for v in api.call('GET', path + '/suites')['items']]
            print(json.dumps(result, indent=2)); return
        config = json.loads(Path(args.config).read_text())
        assert config['projectId'] == args.project, 'Select the exact configured project'
        project = '/api/v1/projects/' + urllib.parse.quote(args.project, safe='')
        projects=api.call('GET','/api/v1/projects')['items']
        if not any(v['id']==args.project for v in projects):
            assert args.action=='configure' and config.get('project',{}).get('id')==args.project, 'The selected project does not exist'
            api.call('POST','/api/v1/projects',config['project'])
        api.call('GET', project)
        receipt_path = Path(args.receipt); receipt_path.parent.mkdir(parents=True, exist_ok=True)
        receipt = json.loads(receipt_path.read_text()) if receipt_path.exists() else {'projectId': args.project, 'rows': {}}
        assert receipt['projectId'] == args.project

        def save():
            receipt_path.write_text(json.dumps(receipt, indent=2) + '\n')

        def ensure_resource(kind, value):
            rows = api.call('GET', project + '/' + kind)['items']
            old = next((v for v in rows if v['id'] == value['id']), None)
            if old:
                assert all(old['data'].get(k) == v for k, v in value.items()), 'Existing ' + kind + ' configuration differs; review it explicitly'
                return old
            return api.call('PUT', project + '/' + kind + '/' + value['id'], value, 0)

        if args.action == 'configure':
            for assignment in config.get('access', []):
                existing=api.call('GET',project+'/role-assignments?limit=100')['items']
                if not any(all(row['data'].get(k)==assignment[k] for k in ['principalKind','principalId','roleId']) for row in existing):
                    api.call('POST',project+'/role-assignments',assignment)
            for app in config['applications']:
                ensure_resource('applications', app)
            ensure_resource('suites', config['suite'])
            for selected in config['probes']:
                row = receipt['rows'].setdefault(selected['id'], {})
                parsed = urllib.parse.urlsplit(selected['url'])
                assert parsed.scheme == 'https' and parsed.hostname and parsed.username is None and parsed.password is None and not parsed.fragment, 'Choose an explicit credential-free HTTPS address'
                origin = parsed.scheme + '://' + parsed.netloc
                env = {'id': selected['environmentId'], 'name': selected['name'] + ' target', 'applicationId': selected['applicationId'], 'stage': selected['stage'], 'protected': selected['protected'], 'endpoints': {'api': origin}}
                ensure_resource('environments', env); row['environmentId'] = env['id']; save()
                definition = {'id': selected['id'], 'title': selected['name'], 'applicationId': selected['applicationId'], 'category': 'Availability/' + selected['stage'], 'type': 'https', 'runner': 'http', 'state': 'enabled', 'safety': selected['safety'], 'rationale': selected['rationale'], 'description': '', 'tags': [], 'dependencies': [], 'requiredBindings': [], 'timeoutSeconds': 10, 'check': {'endpoint': 'api', 'path': parsed.path or '/', 'method': 'GET', 'statusCodes': [200], 'minCertificateDays': 14}}
                assert not parsed.query, 'Configure query-bearing probes through reviewed definition editing'
                test_path = project + '/suites/' + config['suite']['id'] + '/tests'
                current = api.call('GET', test_path + '?limit=100&state=all')['items']
                assert all(v['id'] in [p['id'] for p in config['probes']] for v in current), 'This suite contains other definitions; review shared publication separately'
                old = next((v for v in current if v['id'] == selected['id']), None)
                if old:
                    assert all(old['data'].get(k) == v for k, v in definition.items()), 'Existing test differs; review its definition explicitly'
                else:
                    api.call('POST', test_path, definition)
                row['testSaved'] = True; save()
                binding = api.call('POST', project + '/probes', {'name': selected['name'], 'suiteId': config['suite']['id'], 'testId': selected['id'], 'environmentId': env['id'], 'endpoint': 'api'})
                row['probeId'] = binding['id']; save()
            if not receipt.get('releaseId'):
                path = project + '/suites/' + config['suite']['id'] + '/catalog-drafts'
                draft = api.call('POST', path, {}); validation = api.call('POST', path + '/' + draft['id'] + '/validate', {})
                assert validation['valid'], 'The selected definitions require correction'
                release = api.call('POST', path + '/' + draft['id'] + '/publish', {'activate': True}, draft['revision'], 'probe-release-' + draft['id'])
                receipt['releaseId'] = release['id']; save()
            print(json.dumps({'configured': len(receipt['rows']), 'publishedRelease': receipt['releaseId'], 'runsStarted': 0, 'schedulesEnabled': 0})); return
        if args.action == 'run':
            for selected in config['probes']:
                row = receipt['rows'][selected['id']]
                if not row.get('runId'):
                    run = api.call('POST', project + '/runs', {'suiteId': config['suite']['id'], 'releaseId': receipt['releaseId'], 'environmentId': selected['environmentId'], 'testIds': [selected['id']], 'timeoutSeconds': 60, 'retries': 0}, key='probe-initial-' + row['probeId'] + '-' + receipt['releaseId'])
                    row['runId'] = run['id']; save()
            print(json.dumps({'initialRunIds': [v['runId'] for v in receipt['rows'].values()]})); return
        if args.action == 'enable':
            for selected in config['probes']:
                row = receipt['rows'][selected['id']]
                run = api.call('GET', project + '/runs/' + row['runId'])['data']
                assert run['state'] == 'completed' and run['outcome'] == 'passed', 'Inspect the initial result before enabling its schedule'
                schedules = api.call('GET', project + '/schedules?limit=100')['items']
                schedule_id = 'probe-' + selected['id'].lower()
                old = next((v for v in schedules if v['id'] == schedule_id), None)
                definition = {'id': schedule_id, 'name': selected['name'] + ' checks', 'enabled': True, 'kind': 'interval', 'intervalSeconds': 300, 'timezone': 'UTC', 'overlap': 'skip', 'request': {'suiteId': config['suite']['id'], 'environmentId': selected['environmentId'], 'testIds': [selected['id']], 'executionLocation': 'local', 'timeoutSeconds': 60, 'retries': 0}}
                if not old:
                    api.call('POST', project + '/schedules/preview', definition)
                    api.call('POST', project + '/schedules', definition)
                else:
                    assert old['data']['enabled'] and old['data']['request']['testIds'] == [selected['id']] and old['data']['request']['environmentId'] == selected['environmentId'], 'Existing schedule requires explicit review'
                row['scheduleId'] = schedule_id; save()
            print(json.dumps({'enabledSchedules': [v['scheduleId'] for v in receipt['rows'].values()], 'intervalSeconds': 300})); return
        status = []
        for selected in config['probes']:
            row = receipt['rows'][selected['id']]
            value = api.call('GET', project + '/runs/' + row['runId'])['data']
            item = {'name': selected['name'], 'runId': value['id'], 'state': value['state'], 'outcome': value['outcome'], 'evidence': value['evidenceState']}
            if value.get('results'):
                item.update({'durationMs': value['results'][0]['durationMs'], 'httpStatus': value['results'][0].get('http', {}).get('status'), 'certificateExpiry': value['results'][0].get('http', {}).get('certificateExpires')})
            if row.get('scheduleId'):
                schedule = api.call('GET', project + '/schedules/' + row['scheduleId'])['data']
                item['schedule'] = {k: schedule.get(k) for k in ['enabled', 'nextAt', 'lastAt', 'lastRunId', 'lastState']}
                if schedule.get('lastRunId'):
                    scheduled=api.call('GET',project+'/runs/'+schedule['lastRunId'])['data']
                    item['schedule']['observedState']=scheduled['state']; item['schedule']['observedOutcome']=scheduled['outcome']; item['schedule']['evidence']=scheduled['evidenceState']
            status.append(item)
        receipt['observations'] = status; save(); print(json.dumps(status, indent=2))
    finally:
        if api.csrf:
            assert api.call('POST', '/auth/logout', {})['signedOut'], 'The owned session must be revoked'


if __name__ == '__main__':
    try:
        main()
    except (AssertionError, RuntimeError, urllib.error.URLError) as error:
        raise SystemExit(str(error)) from None
