import argparse
import hashlib
import io
import json
from pathlib import Path
import subprocess
import time
import uuid
import zipfile

from lib.local_kubernetes import LocalAPI
from lib.local_kubernetes_cluster import LocalCluster


TERMINAL = {'completed', 'failed', 'cancelled', 'incomplete', 'timed-out', 'blocked'}


def main():
    parser = argparse.ArgumentParser(description='Check cancellation or controller interruption for an actual local gateway execution.')
    parser.add_argument('--case', choices=['cancel', 'recovery'], required=True)
    args = parser.parse_args()
    installation = Path('.data/local-k3s')
    output = Path('output/local-k3s/gateway/lifecycle')
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    reference_file = output / 'references.json'
    state = json.loads(reference_file.read_text()) if reference_file.exists() else {'suite': 'gateway-lifecycle', 'uploadKey': str(uuid.uuid4()), 'cases': {}}
    credential = (installation / 'gateway/reader').read_bytes()
    cluster = LocalCluster(installation)
    kube = ['kubectl', '--kubeconfig', '/etc/rancher/k3s/k3s.yaml']

    def save(name, value):
        data = json.dumps(value, indent=2).encode()
        assert credential not in data, 'Fixture credentials must stay outside lifecycle diagnostics'
        (output / (name + '.json')).write_bytes(data + b'\n')

    def checkpoint():
        save('references', state)

    def controllers():
        return cluster.pods('teststudio-local', '-l', 'app.kubernetes.io/component=controller,app.kubernetes.io/instance=teststudio')

    def interrupt_owner(run):
        owners = []
        for pod in controllers():
            logs = cluster.execute(*kube, 'logs', pod['metadata']['name'], '-n', 'teststudio-local', '--container=controller', '--tail=200')
            if any('Execution run claimed' in line and run['id'] in line and run['workerId'] in line for line in logs.splitlines()):
                owners.append(pod)
        assert len(owners) == 1, 'The current execution must identify exactly one owning controller'
        owner = owners[0]
        rows = [row for row in owner['status']['containerStatuses'] if row['name'] == 'controller' and row.get('state', {}).get('running')]
        assert len(rows) == 1
        container = rows[0]['containerID'].split('://', 1)[1]
        details = json.loads(cluster.execute('crictl', 'inspect', container))
        labels = details['status']['labels']
        assert labels['io.kubernetes.pod.uid'] == owner['metadata']['uid'] and labels['io.kubernetes.pod.namespace'] == 'teststudio-local' and labels['io.kubernetes.container.name'] == 'controller'
        pid = details['info']['pid']
        assert isinstance(pid, int) and pid > 1 and details['status']['state'] == 'CONTAINER_RUNNING'
        case['interruption'] = {'pod': owner['metadata']['name'], 'uid': owner['metadata']['uid'], 'container': container, 'pid': pid, 'previousRestarts': rows[0]['restartCount']}
        case['interruptionArmed'] = True
        checkpoint()
        cluster.execute('kill', '-KILL', str(pid))
        case['action'] = 'owner-killed'
        checkpoint()

    checkpoint()
    api = LocalAPI(installation)
    api.login()
    base = '/api/v1/projects/k3s-network'
    suite = base + '/suites/' + state['suite']
    try:
        assert len(controllers()) == 2, 'Keep the existing two-controller installation for these cases'
        if not state.get('created'):
            api.call('PUT', suite, {'id': state['suite'], 'name': 'Gateway lifecycle checks', 'applications': ['gateway-fixture'], 'requiresTarget': True}, 0)
            state['created'] = True
            checkpoint()
        if not state.get('preparation'):
            source = Path('tests/kubernetes/environment/lifecycle')
            buffer = io.BytesIO()
            with zipfile.ZipFile(buffer, 'w', zipfile.ZIP_DEFLATED) as archive:
                for file in sorted(source.rglob('*')):
                    if file.is_file():
                        archive.writestr(file.relative_to(source).as_posix(), file.read_bytes())
            prepared = api.call('POST', suite + '/source-bundles?name=gateway-lifecycle.zip&manifest=teststudio-suite.json', buffer.getvalue(), content_type='application/zip', key=state['uploadKey'])
            state['preparation'] = prepared['id']
            checkpoint()
        until = time.monotonic() + 150
        previous = None
        while True:
            prepared = api.call('GET', suite + '/source-preparations/' + state['preparation'])
            save('preparation', prepared)
            job = prepared['data']
            marker = (job['state'], job['phase'])
            if marker != previous:
                print('preparation: ' + '/'.join(marker), flush=True)
                previous = marker
            if job['state'] in {'ready', 'published', 'failed', 'cancelled'}:
                break
            assert time.monotonic() < until, 'Observe the saved preparation rather than creating another'
            time.sleep(1)
        assert job['state'] in {'ready', 'published'}, 'Inspect the retained preparation failure'
        if not state.get('release'):
            upload = api.call('GET', suite + '/source-uploads/' + job['uploadId'])['upload']
            state['release'] = api.call('POST', suite + '/source-uploads/' + upload['id'] + '/publish', {'activate': True}, upload['revision'])['id']
            checkpoint()
        case = state['cases'].setdefault(args.case, {'key': str(uuid.uuid4())})
        checkpoint()
        if not case.get('run'):
            created = api.call('POST', base + '/runs', {'suiteId': state['suite'], 'environmentId': 'api-local', 'testIds': ['NET-GATEWAY-' + args.case.upper() + '-001'], 'timeoutSeconds': 120, 'retries': 0}, key=case['key'])
            case['run'] = created['id']
            checkpoint()
        path = base + '/runs/' + case['run']
        until = time.monotonic() + 180
        previous = None
        while True:
            record = api.call('GET', path)
            execution = api.call('GET', path + '/executions')
            save(args.case + '-run', record)
            save(args.case + '-executions', execution)
            run = record['data']
            assert len(execution['items']) <= 1, 'The selected lifecycle test must keep its original execution identity'
            if execution['items']:
                item = execution['items'][0]['data']
                ref = item['reference']
                if not case.get('execution'):
                    case['execution'], case['worker'], case['gateway'] = ref['id'], ref['container'], ref['gateway']
                    checkpoint()
                assert case['execution'] == ref['id'] and case['worker'] == ref['container'] and case['gateway'] == ref['gateway']
                if not case.get('action') and run.get('currentTestStartedAt'):
                    assert not case.get('interruptionArmed'), 'Inspect the armed original controller interruption before any further action'
                    pods = cluster.pods('teststudio-workers')
                    workers = [pod for pod in pods if pod['metadata']['name'] == ref['container']]
                    gateways = [pod for pod in pods if pod['metadata']['name'] == ref['gateway']]
                    if workers and gateways and any(row.get('state', {}).get('running') for row in workers[0]['status'].get('containerStatuses', [])):
                        container = cluster.worker_runtime(ref['container'], workers[0]['metadata']['uid'])
                        command = ['/usr/local/bin/runsc', '--root=/run/containerd/runsc/k8s.io', 'exec', '--user=1000:1000', container, '/bin/cat', '/output/gateway-lifecycle-ready.json']
                        found = subprocess.run(cluster.command + command, capture_output=True, timeout=10)
                        if found.returncode == 0:
                            marker = json.loads(found.stdout)
                            assert marker == {'testId': ref['testId'], 'connected': True}
                            case['connected'] = True
                            case['workerUid'], case['gatewayUid'] = workers[0]['metadata']['uid'], gateways[0]['metadata']['uid']
                            checkpoint()
                            if args.case == 'cancel':
                                api.call('POST', path + '/cancel', {})
                                case['action'] = 'cancel-requested'
                                checkpoint()
                            else:
                                interrupt_owner(run)
            marker = (run['state'], execution['items'][0]['data']['state'] if execution['items'] else None)
            if marker != previous:
                print(args.case + ': ' + str(marker), flush=True)
                previous = marker
            if run['state'] in TERMINAL and execution['items'] and execution['items'][0]['data']['state'] == 'cleaned':
                break
            assert time.monotonic() < until, 'Observe the same run and gateway bundle; a wait expiry never permits replacement'
            time.sleep(.5)
        assert case.get('action') and case.get('connected'), 'The requested action must follow an actual authenticated gateway request'
        item = execution['items'][0]['data']
        retained = {}
        for kind in ['attempts', 'events', 'logs', 'artifacts']:
            retained[kind] = api.call('GET', path + '/' + kind)
            save(args.case + '-' + kind, retained[kind])
        assert run['counts']['total'] == 1 and len(retained['attempts']['items']) == 1
        observed = retained['attempts']['items'][0]['data']
        if args.case == 'cancel':
            assert run['state'] == 'cancelled' and run['cancelRequested'] and not run['results']
            assert observed['state'] == 'unknown' and observed['failureKind'] == 'cancelled' and observed['durationUnavailable']
        else:
            assert item['recovered'] and item['resultSaved'] and run['state'] == 'incomplete' and run['outcome'] == 'unknown'
            assert observed['state'] == 'passed' and observed['durationMs'] > 0 and item['evidenceComplete']
            owner = case['interruption']
            restarted = [row for pod in controllers() if pod['metadata']['uid'] == owner['uid'] for row in pod['status'].get('containerStatuses', []) if row['name'] == 'controller']
            assert len(restarted) == 1 and restarted[0]['restartCount'] > owner['previousRestarts']
        resources = json.loads(cluster.execute(*kube, 'get', 'pods,services,networkpolicies', '-n', 'teststudio-workers', '-o', 'json'))
        assert not any(row['metadata']['name'] in [case['worker'], case['gateway']] for row in resources['items'])
        artifacts = [row['data'] for row in retained['artifacts']['items'] if row['data']['state'] == 'available']
        assert artifacts, 'The interrupted run must retain its available private diagnostics'
        for artifact in artifacts:
            grant = api.call('POST', path + '/artifacts/' + artifact['id'] + '/access', {'action': 'download'})
            data = api.download(grant['url'])
            assert len(data) == artifact['size'] and hashlib.sha256(data).hexdigest() == artifact['checksum'] and credential not in data
        summary = {'passed': True, 'case': args.case, 'runId': case['run'], 'executionId': case['execution'], 'runState': run['state'], 'attemptState': observed['state'], 'durationMs': observed.get('durationMs'), 'evidenceState': run['evidenceState'], 'privateArtifactsVerified': len(artifacts), 'authenticatedTargetReachedBeforeAction': True, 'singleExecution': True, 'cleaned': True, 'cloudOperations': False, 'hostBrowserStarted': False}
        save(args.case + '-summary', summary)
        print(json.dumps(summary), flush=True)
    finally:
        api.logout()


if __name__ == '__main__':
    main()
