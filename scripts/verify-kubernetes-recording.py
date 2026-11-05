import argparse
import http.server
import json
import os
from pathlib import Path
import signal
import subprocess
import tempfile
import threading
import time
import uuid
from urllib.parse import urlsplit

from lib.local_kubernetes import LocalAPI
from lib.local_kubernetes_cluster import LocalCluster


def processes():
    rows = subprocess.check_output(['ps', '-axo', 'pid=,ppid='], text=True).splitlines()
    return {int(row.split()[0]): int(row.split()[1]) for row in rows if len(row.split()) == 2}


def observe(root, known):
    parents = processes()
    current = {root}
    while True:
        more = {pid for pid, parent in parents.items() if parent in current} - current
        if not more:
            break
        current.update(more)
    known.update(current)
    return known & parents.keys()


def main():
    parser = argparse.ArgumentParser(description='Verify one native desktop recorder handoff through the owned local Kubernetes HTTPS installation')
    parser.add_argument('--retry-after-fix', action='store_true')
    arguments = parser.parse_args()
    root = Path.cwd()
    installation = root / '.data/local-k3s'
    output = root / 'output/local-k3s/recording'
    deployed = json.loads((output / 'installation.json').read_text())
    assert deployed['deployed']
    cluster = LocalCluster(installation)
    assert not cluster.pods('teststudio-workers'), 'Reuse an idle installation without dispatching another suite'
    references = output / 'references.json'
    state = json.loads(references.read_text()) if references.exists() else {'project': 'k3s-recording', 'suite': 'desktop'}

    def checkpoint():
        references.write_text(json.dumps(state, indent=2) + '\n')

    api = LocalAPI(installation)
    api.login()
    native = None
    known = set()
    target = None
    target_thread = None
    private = None
    cleanup = {'remainingProcesses': 0, 'sessionRevoked': False, 'targetClosed': True, 'privateConnectionRemoved': True}
    try:
        base = '/api/v1/projects/' + state['project']
        if not state.get('projectCreated'):
            api.call('POST', '/api/v1/projects', {'id': state['project'], 'name': 'Kubernetes desktop recording', 'description': 'Independent local desktop recorder handoff'})
            state['projectCreated'] = True
            checkpoint()
        suite = base + '/suites/' + state['suite']
        if not state.get('suiteCreated'):
            api.call('PUT', suite, {'id': state['suite'], 'name': 'Recorded desktop flows', 'applications': [], 'requiresTarget': True}, 0)
            state['suiteCreated'] = True
            checkpoint()
        recordings = suite + '/browser-recordings'
        options = api.call('GET', recordings + '/options')
        assert options['enabled'] and options['mode'] == 'companion'
        if state.get('nativeAttempted') and not state.get('nativeCompleted'):
            previous = api.call('GET', recordings + '/' + state['recordingId'])
            assert arguments.retry_after_fix and json.loads((output / 'cleanup.json').read_text())['remainingProcesses'] == 0, 'Observe the original native attempt before retrying'
            assert previous['data']['state'] in ['ready', 'failed', 'cancelled'], 'The original recording must finish before a corrected attempt'
            state.setdefault('previousRecordings', []).append({'id': state.pop('recordingId'), 'state': previous['data']['state']})
            state.update(nativeAttempted=False, stopRequested=False)
            checkpoint()
        if not state.get('nativeCompleted'):
            class Target(http.server.BaseHTTPRequestHandler):
                def do_GET(self):
                    body = b'<!doctype html><title>Kubernetes companion target</title><main><h1>Independent recording target</h1><button>Continue</button></main>'
                    self.send_response(200)
                    self.send_header('Content-Type', 'text/html')
                    self.send_header('Content-Length', str(len(body)))
                    self.end_headers()
                    self.wfile.write(body)

                def log_message(self, *_):
                    pass

            reserved_port = 0
            if state.get('recordingId'):
                previous_address = urlsplit(state['target'])
                assert previous_address.scheme == 'http' and previous_address.hostname == '127.0.0.1'
                reserved_port = previous_address.port
            target = http.server.ThreadingHTTPServer(('127.0.0.1', reserved_port), Target)
            cleanup['targetClosed'] = False
            target_thread = threading.Thread(target=target.serve_forever)
            target_thread.start()
            address = 'http://127.0.0.1:' + str(target.server_port)
            if not state.get('recordingId'):
                state['recordingId'] = 'desktop-' + str(uuid.uuid4())
                state['target'] = address
                checkpoint()
            assert state['target'] == address, 'An uncertain reservation needs review before another target is selected'
            value = {'id': state['recordingId'], 'url': address, 'testId': 'REC-K3S-001', 'title': 'Kubernetes recorded navigation'}
            row = api.call('POST', recordings, value)
            assert row['data']['state'] == 'pairing'
            pairing = api.call('POST', recordings + '/' + state['recordingId'] + '/pairing')
            with tempfile.TemporaryDirectory(prefix='recording-private-', dir=installation) as temporary:
                private = Path(temporary)
                connection = private / 'input.json'
                config = {'connection': pairing['id'] + ':' + pairing['token'], 'certificate': str(installation / 'tls/server.crt'), 'directory': str(private / 'desktop')}
                descriptor = os.open(connection, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
                with os.fdopen(descriptor, 'w') as handle:
                    json.dump(config, handle)
                state['nativeAttempted'] = True
                checkpoint()
                environment = {**os.environ, 'GOWORK': 'off', 'TESTSTUDIO_KUBERNETES_RECORDING_INPUT': str(connection)}
                try:
                    with (output / 'native-companion.log').open('w') as log:
                        native = subprocess.Popen(['go', 'test', './adapters/recording/companion', '-run', '^TestDesktopCompanionAgainstOwnedKubernetes$', '-count=1', '-timeout=80s', '-v'], env=environment, stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
                        state['nativeProcess'] = native.pid
                        checkpoint()
                        deadline = time.monotonic() + 75
                        while native.poll() is None:
                            observe(native.pid, known)
                            captured = api.call('GET', recordings + '/' + state['recordingId'])['data']
                            if captured['state'] == 'recording' and 'page.goto(' in captured.get('code', '') and not state.get('stopRequested'):
                                api.call('POST', recordings + '/' + state['recordingId'] + '/actions/stop')
                                state['stopRequested'] = True
                                checkpoint()
                            assert time.monotonic() < deadline, 'Observe the owned native recorder; do not create another session'
                            time.sleep(0.2)
                        assert native.returncode == 0, 'Inspect the retained native companion log'
                        state['nativeCompleted'] = True
                        checkpoint()
                finally:
                    try:
                        if native and native.poll() is None:
                            try:
                                api.call('POST', recordings + '/' + state['recordingId'] + '/actions/cancel')
                            finally:
                                observe(native.pid, known)
                                try:
                                    native.wait(timeout=20)
                                except subprocess.TimeoutExpired:
                                    os.killpg(native.pid, signal.SIGTERM)
                                    try:
                                        native.wait(timeout=12)
                                    except subprocess.TimeoutExpired:
                                        os.killpg(native.pid, signal.SIGKILL)
                                        native.wait(timeout=5)
                    finally:
                        deadline = time.monotonic() + 10
                        alive = known & processes().keys()
                        while alive and time.monotonic() < deadline:
                            time.sleep(0.1)
                            alive = known & processes().keys()
                        cleanup.update(ownedProcesses=len(known), remainingProcesses=len(alive), remainingProcessIds=sorted(alive))
                        assert not alive, 'Owned recording processes remain after shutdown'
        captured = api.call('GET', recordings + '/' + state['recordingId'])['data']
        assert captured['state'] == 'ready' and 'page.goto(' in captured.get('code', '') and state['target'] in captured['code']
        (output / 'recording.json').write_text(json.dumps(captured, indent=2) + '\n')
        if not state.get('draftId'):
            draft = api.call('GET', recordings + '/' + state['recordingId'] + '/source')
            state['draftId'] = draft['id']
            (output / 'reviewed-draft.json').write_text(json.dumps(draft, indent=2) + '\n')
            checkpoint()
        draft = json.loads((output / 'reviewed-draft.json').read_text())
        source = next(file['content'] for file in draft['files'] if file['path'] == 'checks/recording.spec.ts')
        catalog = json.loads(next(file['content'] for file in draft['files'] if file['path'] == 'catalog.json'))
        assert '@REC-K3S-001' in source and 'TESTSTUDIO_BASE_URL' in source and catalog['tests'][0]['safety'] == 'needs-review'
        if not state.get('draftSaved'):
            api.call('PUT', suite + '/playwright-drafts/' + draft['id'], draft, 0)
            state['draftSaved'] = True
            checkpoint()
        saved = api.call('GET', suite + '/playwright-drafts/' + draft['id'])
        assert saved['revision'] == 1 and saved['data']['files'] == draft['files']
        assert api.call('GET', base + '/runs')['items'] == []
        for project, expected in deployed['previousRuns'].items():
            assert sorted(row['id'] for row in api.call('GET', '/api/v1/projects/' + project + '/runs')['items']) == expected
        state['verified'] = True
        checkpoint()
        summary = {'passed': True, 'helmRevision': deployed['helmRevision'], 'recordingId': state['recordingId'], 'recordingState': captured['state'], 'draftId': draft['id'], 'draftRevision': saved['revision'], 'targetRuns': 0, 'existingRunsUnchanged': {key: len(value) for key, value in deployed['previousRuns'].items()}, 'httpsCertificateVerification': True, 'hostnameVerification': True, 'normalOIDC': True, 'cloudOperations': False}
        (output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
        print(json.dumps(summary), flush=True)
    finally:
        if target:
            target.shutdown()
            target.server_close()
            target_thread.join(timeout=2)
            cleanup['targetClosed'] = not target_thread.is_alive()
        cleanup['privateConnectionRemoved'] = private is None or not private.exists()
        if not cleanup.get('ownedProcesses') and (output / 'cleanup.json').exists():
            previous_cleanup = json.loads((output / 'cleanup.json').read_text())
            cleanup.update({key: previous_cleanup[key] for key in ['ownedProcesses', 'remainingProcesses', 'privateConnectionRemoved'] if key in previous_cleanup})
        try:
            api.logout()
            cleanup['sessionRevoked'] = True
        finally:
            (output / 'cleanup.json').write_text(json.dumps(cleanup, indent=2) + '\n')


if __name__ == '__main__':
    main()
