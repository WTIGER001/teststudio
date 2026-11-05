import datetime
import json
from pathlib import Path
import time
import sys
import urllib.error
import uuid

from lib.local_kubernetes import LocalAPI
from lib.local_kubernetes_cluster import LocalCluster


def main():
    output = Path('output/local-k3s/deadline')
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    state_file = output / 'references.json'
    state = json.loads(state_file.read_text()) if state_file.exists() else {'key': str(uuid.uuid4())}

    def save(name, data):
        (output / (name + '.json')).write_text(json.dumps(data, indent=2) + '\n')

    def checkpoint():
        save('references', state)

    checkpoint()
    api = LocalAPI(Path('.data/local-k3s'))
    cluster = LocalCluster()
    api.login()
    try:
        if not state.get('run'):
            request = {'suiteId': 'lifecycle-local', 'testIds': ['K3S-RECOVERY-001'], 'timeoutSeconds': 15, 'retries': 0}
            state['run'] = api.call('POST', '/api/v1/projects/k3s-coded/runs', request, key=state['key'])['id']
            checkpoint()
        path = '/api/v1/projects/k3s-coded/runs/' + state['run']
        if not state.get('terminationVerified'):
            assert not state.get('pauseArmed'), 'Inspect the existing paused processes and run; do not issue another pause or create a replacement'
            until = time.monotonic() + 35
            while True:
                record = api.call('GET', path)
                save('run-before-pause', record)
                executions = api.call('GET', path + '/executions')
                save('executions-before-pause', executions)
                if record['data'].get('currentTestStartedAt') and executions['items']:
                    break
                assert record['data']['state'] not in {'completed', 'incomplete', 'timed-out', 'cancelled', 'blocked'}, 'The original run ended before the pause window'
                assert time.monotonic() < until
                time.sleep(.2)
            assert len(executions['items']) == 1
            ref = executions['items'][0]['data']['reference']
            pod = cluster.worker_status(ref['container'])
            assert pod['phase'] == 'Running' and pod['activeDeadlineSeconds'] > 1
            timestamp = record['data']['deadline']
            assert timestamp.endswith('Z')
            whole, fraction_separator, fraction = timestamp[:-1].partition('.')
            timestamp = whole + ('.' + fraction[:6] if fraction_separator else '') + '+00:00'
            deadline = datetime.datetime.fromisoformat(timestamp)
            remaining = (deadline - datetime.datetime.now(datetime.timezone.utc)).total_seconds()
            assert remaining > 3, 'The existing worker has insufficient time for an independent deadline observation'
            state.update(execution=ref['id'], worker=pod, workerContainer=cluster.worker_runtime(pod['name'], pod['uid']), controller=cluster.api_process(), pauseArmed=True)
            checkpoint()
            try:
                cluster.signal_worker(state['workerContainer'], 'STOP')
                runtime = cluster.runtime_state(state['workerContainer'])
                save('guest-stop-runtime', runtime)
                assert runtime['status'] == 'running', 'The runtime must remain available for Kubernetes control'
                status = cluster.worker_process_status(state['workerContainer'])
                (output / 'stopped-supervisor.txt').write_text(status)
                assert any(line.startswith('State:') and 'T' in line for line in status.splitlines()), 'The guest supervisor must actually be stopped'
                while (deadline - datetime.datetime.now(datetime.timezone.utc)).total_seconds() > 2:
                    time.sleep(.1)
                cluster.signal_api(state['controller'], 'STOP')
                assert cluster.process_state(state['controller']) == 'T'
                state['controllerPaused'] = True
                checkpoint()
                until = time.monotonic() + 20
                while True:
                    pod = cluster.worker_status(ref['container'])
                    save('provider-termination', pod)
                    if pod.get('phase') == 'Failed' and pod.get('containers') and all(row.get('state', {}).get('terminated') for row in pod['containers']):
                        break
                    assert not pod.get('absent'), 'Worker absence alone cannot prove termination'
                    assert time.monotonic() < until, 'The original independent deadline remains unconfirmed'
                    time.sleep(.25)
                assert pod['reason'] == 'DeadlineExceeded', 'Kubernetes must report its own deadline termination'
                assert pod['uid'] == state['worker']['uid'] and pod['activeDeadlineSeconds'] == state['worker']['activeDeadlineSeconds']
                assert cluster.process_state(state['controller']) == 'T', 'The controller must remain paused until worker termination'
                state['terminationVerified'] = True
                checkpoint()
            finally:
                try:
                    if state.get('controller'):
                        cluster.signal_api(state['controller'], 'CONT')
                        state['controllerResumed'] = True
                        checkpoint()
                finally:
                    provider = cluster.worker_status(state['worker']['name'])
                    if not provider.get('absent') and provider.get('phase') == 'Running':
                        runtime = cluster.runtime_state(state['workerContainer'])
                        if runtime['status'] == 'running':
                            cluster.signal_worker(state['workerContainer'], 'CONT')
                    state['pauseCleanupFinished'] = True
                    checkpoint()
        until = time.monotonic() + 110
        while True:
            try:
                record = api.call('GET', path)
            except (urllib.error.URLError, RuntimeError) as error:
                if time.monotonic() >= until or isinstance(error, RuntimeError) and ': 503 ' not in str(error) and ': 502 ' not in str(error):
                    raise
                time.sleep(.5)
                continue
            executions = api.call('GET', path + '/executions')
            save('run', record)
            save('executions', executions)
            print(record['data']['state'] + '/' + executions['items'][0]['data']['state'], flush=True)
            assert len(executions['items']) == 1 and executions['items'][0]['id'] == state['execution']
            if executions['items'][0]['data']['state'] == 'cleaned':
                break
            assert time.monotonic() < until, 'The same original execution remains pending cleanup'
            time.sleep(2)
        for kind in ['attempts', 'events', 'artifacts', 'logs']:
            save(kind, api.call('GET', path + '/' + kind))
        run = record['data']
        assert run['state'] == 'timed-out' and run['outcome'] == 'timed-out'
        assert len(run['results']) == 1 and run['results'][0]['state'] == 'timed-out' and run['results'][0]['durationUnavailable']
        assert cluster.worker_status(state['worker']['name']).get('absent')
        summary = {'passed': True, 'runId': state['run'], 'executionId': state['execution'], 'guestProcessesStopped': True, 'runtimeControlAvailable': True, 'controllerPausedThroughTermination': True, 'originalPodDeadlineUnchanged': True, 'workerTerminationVerified': True, 'controllerResumed': state['controllerResumed'], 'cleaned': True, 'durationInferred': False, 'cloudOperations': False}
        save('summary', summary)
        print(json.dumps(summary), flush=True)
    except BaseException as error:
        save('verification-error', {'type': type(error).__name__, 'message': str(error)})
        raise
    finally:
        primary = sys.exc_info()[0]
        try:
            api.logout()
        except Exception:
            save('logout-pending', {'pending': True})
            if primary is None:
                raise


if __name__ == '__main__':
    main()
