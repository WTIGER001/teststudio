import hashlib
import json
from pathlib import Path
import subprocess
import time
import uuid

from lib.local_kubernetes import LocalAPI
from lib.local_kubernetes_cluster import LocalCluster


def main():
    root = Path('output/local-k3s/controllers')
    root.mkdir(parents=True, exist_ok=True, mode=0o700)
    state_file = root / 'failover-reference.json'
    state = json.loads(state_file.read_text()) if state_file.exists() else {'key': str(uuid.uuid4())}
    cluster = LocalCluster()
    kube = ['kubectl', '--kubeconfig', '/etc/rancher/k3s/k3s.yaml']
    selector = 'app.kubernetes.io/component=controller,app.kubernetes.io/instance=teststudio'

    def save(name, value):
        (root / (name + '.json')).write_text(json.dumps(value, indent=2) + '\n')

    def checkpoint():
        save('failover-reference', state)

    def controllers():
        return cluster.pods('teststudio-local', '-l', selector)

    def ready(pod):
        return not pod['metadata'].get('deletionTimestamp') and any(row['type'] == 'Ready' and row['status'] == 'True' for row in pod['status'].get('conditions', []))

    def receipt(pod):
        rows = [row for row in pod['status'].get('containerStatuses', []) if row['name'] == 'controller']
        assert len(rows) == 1 and rows[0].get('state', {}).get('running')
        row = rows[0]
        identifier = row['containerID'].split('://', 1)[1]
        details = json.loads(cluster.execute('crictl', 'inspect', identifier))
        labels = details['status']['labels']
        assert labels['io.kubernetes.pod.uid'] == pod['metadata']['uid'] and labels['io.kubernetes.pod.namespace'] == 'teststudio-local' and labels['io.kubernetes.container.name'] == 'controller'
        pid = details['info']['pid']
        assert isinstance(pid, int) and pid > 1
        return {'name': pod['metadata']['name'], 'uid': pod['metadata']['uid'], 'containerId': identifier, 'pid': pid, 'restarts': row['restartCount']}

    def signal(owner, value):
        assert value in {'STOP', 'KILL', 'CONT'}
        details = json.loads(cluster.execute('crictl', 'inspect', owner['containerId']))
        assert details['status']['labels']['io.kubernetes.pod.uid'] == owner['uid'] and details['status']['labels']['io.kubernetes.container.name'] == 'controller'
        assert details['info']['pid'] == owner['pid'] and details['status']['state'] == 'CONTAINER_RUNNING'
        cluster.execute('kill', '-' + value, str(owner['pid']))

    def scale(replicas):
        cluster.execute(*kube, 'scale', 'deployment/teststudio-controller', '-n', 'teststudio-local', '--replicas=' + str(replicas))

    checkpoint()
    api = LocalAPI(Path('.data/local-k3s'))
    api.login()
    try:
        pods = controllers()
        if not state.get('interrupted') or state.get('restored'):
            assert len(pods) == 2 and all(ready(pod) for pod in pods), 'The two-controller installation must be Ready before this case'
        assert not state.get('restored') or state.get('peerRecoveryVerified'), 'The pool was restored before peer-only recovery was proved; inspect the saved original case'
        api_pods = cluster.pods('teststudio-local', '-l', 'app.kubernetes.io/component=api')
        assert len(api_pods) == 1 and '--execution-worker=false' in api_pods[0]['spec']['containers'][0]['args']
        if not state.get('retainedEvidenceVerified'):
            old = json.loads(Path('output/local-k3s/coded/references.json').read_text())['suites']['playwright']['run']
            path = '/api/v1/projects/k3s-coded/runs/' + old
            artifacts = api.call('GET', path + '/artifacts')['items']
            image = next(row['data'] for row in artifacts if row['data']['contentType'] == 'image/png' and row['data']['state'] == 'available')
            grant = api.call('POST', path + '/artifacts/' + image['id'] + '/access', {'action': 'download'})
            body = api.download(grant['url'])
            assert len(body) == image['size'] and hashlib.sha256(body).hexdigest() == image['checksum']
            state['retainedEvidenceVerified'] = {'run': old, 'artifact': image['id'], 'checksum': image['checksum']}
            checkpoint()
        if not state.get('run'):
            request = {'suiteId': 'lifecycle-local', 'testIds': ['K3S-RECOVERY-001'], 'timeoutSeconds': 120, 'retries': 0}
            state['run'] = api.call('POST', '/api/v1/projects/k3s-coded/runs', request, key=state['key'])['id']
            checkpoint()
        path = '/api/v1/projects/k3s-coded/runs/' + state['run']
        if not state.get('interrupted'):
            assert not state.get('stopArmed'), 'Inspect the saved original controller; do not repeat an armed interruption'
            until = time.monotonic() + 80
            while True:
                run = api.call('GET', path)
                executions = api.call('GET', path + '/executions')
                save('run-before-interruption', run)
                save('executions-before-interruption', executions)
                if run['data'].get('currentTestStartedAt') and executions['items']:
                    break
                assert run['data']['state'] not in {'completed', 'incomplete', 'timed-out', 'cancelled', 'blocked'}, 'Inspect the existing run before creating any replacement'
                assert time.monotonic() < until, 'Inspect this same run and execution; observation expiry does not authorize another'
                time.sleep(.5)
            assert len(executions['items']) == 1
            state['execution'] = executions['items'][0]['id']
            state['workerPod'] = executions['items'][0]['data']['reference']['container']
            claimed = []
            for pod in controllers():
                text = cluster.execute(*kube, 'logs', pod['metadata']['name'], '-n', 'teststudio-local', '--container=controller', '--tail=200')
                lines = [line for line in text.splitlines() if 'Execution run claimed' in line and state['run'] in line and run['data']['workerId'] in line]
                if lines:
                    claimed.append(pod)
                    (root / 'owner-claim.log').write_text('\n'.join(lines) + '\n')
            assert len(claimed) == 1, 'The successful claim must identify exactly one controller'
            owner = claimed[0]
            peers = [pod for pod in controllers() if pod['metadata']['uid'] != owner['metadata']['uid']]
            assert len(peers) == 1 and ready(peers[0])
            state.update(owner=receipt(owner), peer=receipt(peers[0]), stopArmed=True)
            checkpoint()
            signal(state['owner'], 'STOP')
            state['ownerStopped'] = True
            checkpoint()
            for pod, cost in [(owner, '-1000'), (peers[0], '1000')]:
                patch = {'metadata': {'uid': pod['metadata']['uid'], 'annotations': {'controller.kubernetes.io/pod-deletion-cost': cost}}}
                cluster.execute(*kube, 'patch', 'pod', pod['metadata']['name'], '-n', 'teststudio-local', '--type=merge', '-p', json.dumps(patch))
            scale(1)
            state['scaledForIsolation'] = True
            checkpoint()
            until = time.monotonic() + 10
            while True:
                current = controllers()
                removed = [pod for pod in current if pod['metadata']['uid'] == state['owner']['uid']]
                peers = [pod for pod in current if pod['metadata']['uid'] == state['peer']['uid'] and ready(pod)]
                assert len(peers) == 1, 'The surviving peer must remain Ready'
                if removed and removed[0]['metadata'].get('deletionTimestamp'):
                    break
                assert time.monotonic() < until, 'The selected owner was not the scale-down target'
                time.sleep(.2)
            signal(state['owner'], 'KILL')
            state.update(interrupted=True, ownerKilled=True)
            checkpoint()
        until = time.monotonic() + 150
        previous = None
        while True:
            run = api.call('GET', path)
            executions = api.call('GET', path + '/executions')
            save('run', run)
            save('executions', executions)
            assert len(executions['items']) == 1 and executions['items'][0]['id'] == state['execution']
            if not state.get('restored'):
                peers = [pod for pod in controllers() if pod['metadata']['uid'] == state['peer']['uid']]
                assert len(peers) == 1 and ready(peers[0]) and receipt(peers[0]) == state['peer'], 'The original peer must remain live without restart'
                active = [pod for pod in controllers() if not pod['metadata'].get('deletionTimestamp')]
                assert len(active) == 1 and active[0]['metadata']['uid'] == state['peer']['uid']
            marker = (run['data']['state'], executions['items'][0]['data']['state'])
            if marker != previous:
                print(str(marker), flush=True)
                previous = marker
            if marker[1] == 'cleaned':
                break
            assert time.monotonic() < until, 'The same execution remains pending; inspect its saved identity rather than resubmitting'
            time.sleep(1)
        item = executions['items'][0]['data']
        assert item['recovered'] and item['resultSaved'] and item['evidenceComplete']
        assert item['recoveredResult']['state'] == 'passed' and item['recoveredResult']['durationMs'] > 0
        assert run['data']['state'] == 'incomplete' and run['data']['outcome'] == 'unknown'
        assert cluster.worker_status(state['workerPod']).get('absent')
        artifacts = api.call('GET', path + '/artifacts')
        save('artifacts', artifacts)
        artifact = next(row['data'] for row in artifacts['items'] if row['data']['filename'] == 'go-events.jsonl' and row['data']['state'] == 'available')
        grant = api.call('POST', path + '/artifacts/' + artifact['id'] + '/access', {'action': 'download'})
        body = api.download(grant['url'])
        assert len(body) == artifact['size'] and hashlib.sha256(body).hexdigest() == artifact['checksum']
        (root / 'recovered-private-evidence.jsonl').write_bytes(body)
        for kind in ['attempts', 'events', 'logs']:
            save(kind, api.call('GET', path + '/' + kind))
        state['peerRecoveryVerified'] = True
        checkpoint()
        summary = {'passed': True, 'runId': state['run'], 'executionId': state['execution'], 'ownerPodUid': state['owner']['uid'], 'survivingPeerPodUid': state['peer']['uid'], 'peerOnlyDuringRecovery': True, 'peerDidNotRestart': True, 'singleExecution': True, 'recoveredObservation': item['recoveredResult']['state'], 'durationMs': item['recoveredResult']['durationMs'], 'runState': run['data']['state'], 'privateEvidenceVerified': True, 'oldEvidenceRetained': True, 'cleaned': True, 'cloudOperations': False, 'hostBrowserStarted': False}
        save('summary', summary)
        print(json.dumps(summary), flush=True)
    finally:
        try:
            if state.get('ownerStopped') and not state.get('ownerKilled'):
                signal(state['owner'], 'CONT')
                state['ownerResumed'] = True
                checkpoint()
        finally:
            try:
                if state.get('scaledForIsolation') and not state.get('restored'):
                    scale(2)
                    state['restored'] = True
                    checkpoint()
            finally:
                api.logout()


if __name__ == '__main__':
    main()
