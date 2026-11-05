import argparse
import hashlib
import io
import json
from pathlib import Path
import time
import urllib.error
import uuid
import zipfile

from lib.local_kubernetes import LocalAPI
from lib.local_kubernetes_cluster import LocalCluster


TERMINAL = {'completed', 'failed', 'cancelled', 'incomplete', 'timed-out', 'blocked'}


def main():
    parser = argparse.ArgumentParser(description='Verify bounded actual lifecycle cases on the owned local Kubernetes installation.')
    parser.add_argument('--case', choices=['cancel', 'timeout', 'recovery', 'rerun-timeout'], required=True)
    args = parser.parse_args()
    output = Path('output/local-k3s/lifecycle')
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    checkpoint_file = output / 'references.json'
    state = json.loads(checkpoint_file.read_text()) if checkpoint_file.exists() else {'suite': 'lifecycle-local', 'uploadKey': str(uuid.uuid4()), 'cases': {}}

    def save(name, value):
        (output / (name + '.json')).write_text(json.dumps(value, indent=2) + '\n')

    def checkpoint():
        save('references', state)

    checkpoint()
    api = LocalAPI(Path('.data/local-k3s'))
    cluster = LocalCluster()
    api.login()
    suite = '/api/v1/projects/k3s-coded/suites/' + state['suite']
    try:
        if not state.get('created'):
            api.call('PUT', suite, {'id': state['suite'], 'name': 'Local Kubernetes lifecycle', 'applications': ['workers'], 'requiresTarget': False}, 0)
            state['created'] = True
            checkpoint()
        if not state.get('preparation'):
            root = Path('tests/kubernetes/suites/lifecycle')
            buffer = io.BytesIO()
            with zipfile.ZipFile(buffer, 'w', zipfile.ZIP_DEFLATED) as archive:
                for file in sorted(root.rglob('*')):
                    if file.is_file():
                        archive.writestr(file.relative_to(root).as_posix(), file.read_bytes())
            prepared = api.call('POST', suite + '/source-bundles?name=lifecycle.zip&manifest=teststudio-suite.json', buffer.getvalue(), content_type='application/zip', key=state['uploadKey'])
            state['preparation'] = prepared['id']
            checkpoint()
        until = time.monotonic() + 150
        previous = None
        while True:
            preparation = api.call('GET', suite + '/source-preparations/' + state['preparation'])
            save('preparation', preparation)
            job = preparation['data']
            if job['phase'] != previous:
                print('preparation: ' + job['state'] + '/' + job['phase'], flush=True)
                previous = job['phase']
            if job['state'] in {'ready', 'published', 'failed', 'cancelled'}:
                break
            assert time.monotonic() < until, 'Inspect the saved preparation; do not create a replacement'
            time.sleep(1)
        assert job['state'] in {'ready', 'published'}, job.get('error', '')
        if not state.get('release'):
            upload = api.call('GET', suite + '/source-uploads/' + job['uploadId'])['upload']
            state['release'] = api.call('POST', suite + '/source-uploads/' + upload['id'] + '/publish', {'activate': True}, upload['revision'])['id']
            checkpoint()
        case = state['cases'].setdefault(args.case, {'key': str(uuid.uuid4())})
        checkpoint()
        if not case.get('run'):
            if args.case == 'rerun-timeout':
                parent = state['cases']['timeout']['run']
                case['parent'] = parent
                checkpoint()
                created = api.call('POST', '/api/v1/projects/k3s-coded/runs/' + parent + '/rerun-failures', {}, key=case['key'])
            else:
                request = {'suiteId': state['suite'], 'testIds': ['K3S-' + args.case.upper() + '-001'], 'timeoutSeconds': 120, 'retries': 0}
                created = api.call('POST', '/api/v1/projects/k3s-coded/runs', request, key=case['key'])
            case['run'] = created['id']
            checkpoint()
        path = '/api/v1/projects/k3s-coded/runs/' + case['run']
        until = time.monotonic() + 180
        previous = None
        while True:
            try:
                record = api.call('GET', path)
            except (urllib.error.URLError, ConnectionError, TimeoutError, RuntimeError) as error:
                if not case.get('interruptionArmed') or time.monotonic() >= until:
                    raise
                print('Waiting for the same controller/run after interruption: ' + type(error).__name__, flush=True)
                time.sleep(2)
                continue
            save(args.case + '-run', record)
            run = record['data']
            execution = api.call('GET', path + '/executions')
            save(args.case + '-executions', execution)
            if execution['items']:
                assert len(execution['items']) == 1, 'The selected test created duplicate execution records'
                ref = execution['items'][0]['data']['reference']
                provider = cluster.worker_status(ref['container'])
                save(args.case + '-provider', provider)
                history_file = output / (args.case + '-provider-history.json')
                history = json.loads(history_file.read_text()) if history_file.exists() else []
                if not history or history[-1] != provider:
                    history.append(provider)
                    save(args.case + '-provider-history', history)
                case.setdefault('execution', ref['id'])
                assert case['execution'] == ref['id'], 'Recovery replaced the execution identity'
            marker = (run['state'], run['phase'], execution['items'][0]['data']['state'] if execution['items'] else None)
            if marker != previous:
                print(args.case + ': ' + str(marker), flush=True)
                previous = marker
            if run.get('currentTestStartedAt') and not case.get('action'):
                if args.case == 'cancel':
                    api.call('POST', path + '/cancel', {})
                    case['action'] = 'cancel-requested'
                elif args.case == 'recovery':
                    assert not case.get('interruptionArmed'), 'An interruption was already armed; inspect the existing controller before issuing another signal'
                    case['interruptionArmed'] = True
                    checkpoint()
                    case['interruption'] = cluster.interrupt_api()
                    case['action'] = 'controller-killed'
                else:
                    case['action'] = 'await-test-timeout'
                checkpoint()
            cleaned = bool(execution['items']) and execution['items'][0]['data']['state'] == 'cleaned'
            if run['state'] in TERMINAL and cleaned:
                break
            assert time.monotonic() < until, 'The same run or cleanup is pending; inspect saved records without replacement'
            time.sleep(1)
        retained = {}
        for kind in ['attempts', 'events', 'logs', 'artifacts']:
            retained[kind] = api.call('GET', path + '/' + kind)
            save(args.case + '-' + kind, retained[kind])
        assert run['counts']['total'] == 1 and len(run['results']) <= 1
        assert len(retained['attempts']['items']) == 1, 'An interruption must retain one observed attempt'
        observed = retained['attempts']['items'][0]['data']
        if args.case == 'cancel':
            assert run['state'] == 'cancelled' and run['cancelRequested'], 'Cancellation did not produce its expected run state'
            assert observed['state'] == 'unknown' and observed['failureKind'] == 'cancelled' and observed['durationUnavailable']
            assert not run['results'] and run['counts']['completed'] == 0, 'An interrupted test must not invent a final observation'
        elif args.case in {'timeout', 'rerun-timeout'}:
            assert run['outcome'] == 'timed-out' and run['counts']['timedOut'] == 1 and run['counts']['failed'] == 0
            assert run['results'][0]['state'] == 'timed-out' and run['results'][0]['failureKind'] == 'timeout'
            if args.case == 'rerun-timeout':
                assert run['request']['parentRunId'] == case['parent'] and [test['id'] for test in run['tests']] == ['K3S-TIMEOUT-001']
                assert execution['items'][0]['data']['cleanupAttempts'] == 1, 'Acknowledged deletion unexpectedly required a recovery retry'
        else:
            assert case.get('action') == 'controller-killed' and execution['items'][0]['data']['recovered']
            assert run['state'] == 'incomplete' and run['outcome'] == 'unknown' and run['evidenceState'] == 'incomplete'
            assert observed['state'] == 'passed' and execution['items'][0]['data']['evidenceComplete']
            if not case.get('restartVerification'):
                active = cluster.pods('teststudio-local', '-l', 'app.kubernetes.io/name=teststudio')
                restarted = [(pod, row) for pod in active for row in pod['status'].get('containerStatuses', []) if row['name'] == 'api']
                assert len(restarted) == 1 and restarted[0][0]['metadata']['uid'] == case['interruption']['uid']
                assert restarted[0][1]['restartCount'] > case['interruption']['previousRestarts']
                case['restartVerification'] = {'podUid': case['interruption']['uid'], 'restarts': restarted[0][1]['restartCount']}
                checkpoint()
        assert cluster.worker_status(ref['container']).get('absent'), 'The owned worker remains after cleaned status'
        available = [row['data'] for row in retained['artifacts']['items'] if row['data']['state'] == 'available']
        assert available, 'Expected private lifecycle evidence is unavailable'
        artifact = next((item for item in available if item['filename'] == 'go-events.jsonl'), available[0])
        grant = api.call('POST', path + '/artifacts/' + artifact['id'] + '/access', {'action': 'download'})
        body = api.download(grant['url'])
        assert len(body) == artifact['size'] and hashlib.sha256(body).hexdigest() == artifact['checksum']
        (output / (args.case + '-private-evidence')).write_bytes(body)
        summary = {'passed': True, 'case': args.case, 'runId': case['run'], 'executionId': case['execution'], 'runState': run['state'], 'outcome': run['outcome'], 'resultState': run['results'][0]['state'] if run['results'] else None, 'attemptState': observed['state'], 'evidenceState': run['evidenceState'], 'singleExecution': True, 'cleaned': True, 'workerRemoved': True, 'cloudOperations': False, 'privateDownloadVerified': True}
        save(args.case + '-summary', summary)
        print(json.dumps(summary), flush=True)
    finally:
        api.logout()


if __name__ == '__main__':
    main()
