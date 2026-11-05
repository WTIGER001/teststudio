import json
import subprocess
from pathlib import Path


class LocalCluster:
    def __init__(self, installation=Path('.data/local-k3s')):
        ownership = json.loads((installation / 'ownership.json').read_text())
        assert ownership['cluster'] == 'teststudio-local' and ownership['profile'] == 'teststudio-k3s'
        self.command = ['docker', '--host', 'unix:///Users/johnbauer/.colima/teststudio-k3s/docker.sock', 'exec', 'k3d-teststudio-local-server-0']

    def execute(self, *args):
        return subprocess.check_output(self.command + list(args), timeout=20, text=True)

    def pods(self, namespace, *args):
        return json.loads(self.execute('kubectl', '--kubeconfig', '/etc/rancher/k3s/k3s.yaml', 'get', 'pods', '-n', namespace, *args, '-o', 'json'))['items']

    def interrupt_api(self):
        pods = self.pods('teststudio-local', '-l', 'app.kubernetes.io/name=teststudio')
        candidates = [(pod, row) for pod in pods for row in pod['status'].get('containerStatuses', []) if row['name'] == 'api' and row.get('state', {}).get('running')]
        assert len(candidates) == 1, 'Exactly one owned controller must be running'
        pod, container = candidates[0]
        identifier = container['containerID'].split('://', 1)[1]
        details = json.loads(self.execute('crictl', 'inspect', identifier))
        labels = details['status']['labels']
        assert labels['io.kubernetes.pod.namespace'] == 'teststudio-local' and labels['io.kubernetes.pod.uid'] == pod['metadata']['uid'] and labels['io.kubernetes.container.name'] == 'api'
        pid = details['info']['pid']
        assert isinstance(pid, int) and pid > 1
        self.execute('kill', '-KILL', str(pid))
        return {'pod': pod['metadata']['name'], 'uid': pod['metadata']['uid'], 'containerId': identifier, 'previousRestarts': container['restartCount'], 'signal': 'SIGKILL'}

    def worker_status(self, container):
        found = [pod for pod in self.pods('teststudio-workers') if pod['metadata']['name'] == container]
        assert len(found) <= 1
        if not found:
            return {'name': container, 'absent': True}
        pod = found[0]
        return {'name': container, 'uid': pod['metadata']['uid'], 'activeDeadlineSeconds': pod['spec']['activeDeadlineSeconds'], 'phase': pod['status']['phase'], 'reason': pod['status'].get('reason'), 'containers': pod['status'].get('containerStatuses', []), 'conditions': pod['status'].get('conditions', [])}


    def api_process(self):
        pods = self.pods('teststudio-local', '-l', 'app.kubernetes.io/component=api')
        candidates = [(pod, row) for pod in pods for row in pod['status'].get('containerStatuses', []) if row['name'] == 'api' and row.get('state', {}).get('running')]
        assert len(candidates) == 1, 'Exactly one owned API must be running for deadline verification'
        pod, row = candidates[0]
        assert not self.pods('teststudio-local', '-l', 'app.kubernetes.io/component=controller'), 'This deadline fixture requires the single combined controller profile'
        assert '--execution-worker=false' not in pod['spec']['containers'][0].get('args', [])
        container = row['containerID'].split('://', 1)[1]
        details = json.loads(self.execute('crictl', 'inspect', container))
        receipt = {'pod': pod['metadata']['name'], 'uid': pod['metadata']['uid'], 'containerId': container, 'pid': details['info']['pid'], 'previousRestarts': row['restartCount']}
        self.validate_process(receipt)
        return receipt

    def validate_process(self, receipt):
        details = json.loads(self.execute('crictl', 'inspect', receipt['containerId']))
        labels = details['status']['labels']
        assert labels['io.kubernetes.pod.namespace'] == 'teststudio-local' and labels['io.kubernetes.pod.uid'] == receipt['uid'] and labels['io.kubernetes.container.name'] == 'api'
        assert details['info']['pid'] == receipt['pid'] and isinstance(receipt['pid'], int) and receipt['pid'] > 1
        assert details['status']['state'] == 'CONTAINER_RUNNING', 'The original API container is no longer running'

    def signal_api(self, receipt, signal):
        assert signal in {'STOP', 'CONT'}
        self.validate_process(receipt)
        self.execute('kill', '-' + signal, str(receipt['pid']))

    def process_state(self, receipt):
        self.validate_process(receipt)
        status = self.execute('cat', '/proc/' + str(receipt['pid']) + '/stat')
        return status.rsplit(')', 1)[1].strip().split()[0]

    def worker_runtime(self, name, uid):
        pods = [pod for pod in self.pods('teststudio-workers') if pod['metadata']['name'] == name]
        assert len(pods) == 1 and pods[0]['metadata']['uid'] == uid
        statuses = pods[0]['status'].get('containerStatuses', [])
        assert len(statuses) == 1 and statuses[0]['state'].get('running')
        container = statuses[0]['containerID'].split('://', 1)[1]
        details = json.loads(self.execute('crictl', 'inspect', container))
        labels = details['status']['labels']
        assert labels['io.kubernetes.pod.namespace'] == 'teststudio-workers' and labels['io.kubernetes.pod.uid'] == uid and labels['io.kubernetes.container.name'] == 'runner'
        return container

    def runtime_state(self, container):
        assert len(container) == 64 and all(c in '0123456789abcdef' for c in container)
        return json.loads(self.execute('/usr/local/bin/runsc', '--root=/run/containerd/runsc/k8s.io', 'state', container))

    def signal_worker(self, container, signal):
        assert signal in {'STOP', 'CONT'} and len(container) == 64 and all(c in '0123456789abcdef' for c in container)
        return self.execute('/usr/local/bin/runsc', '--root=/run/containerd/runsc/k8s.io', 'kill', '--all', container, signal)

    def worker_process_status(self, container):
        assert len(container) == 64 and all(c in '0123456789abcdef' for c in container)
        return self.execute('/usr/local/bin/runsc', '--root=/run/containerd/runsc/k8s.io', 'exec', '--user=1000:1000', container, '/bin/cat', '/proc/1/status')
