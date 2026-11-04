import argparse
import json
import os
from pathlib import Path
import signal
import socket
import sqlite3
import subprocess
import tempfile
import time
import urllib.error
import urllib.request


def main():
    parser = argparse.ArgumentParser(description='Check the optional private worker listener in one fresh local native service without a browser or cloud target.')
    parser.add_argument('--build', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    build = json.loads(args.build.read_text())
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    held = []
    for _ in range(3):
        item = socket.socket()
        item.bind(('127.0.0.1', 0))
        held.append(item)
    api_port, artifact_port, worker_port = [item.getsockname()[1] for item in held]
    for item in held:
        item.close()
    summary = {'passed': False, 'cloudOperations': False, 'browserRuns': False, 'checks': []}
    cleanup = {'processStopped': False, 'temporaryDataRemoved': False}
    with tempfile.TemporaryDirectory(prefix='listener-', dir=output) as temporary:
        root = Path(temporary)
        with (root / 'service.log').open('w') as log:
            database = root / 'teststudio.db'
            api = 'http://127.0.0.1:' + str(api_port)
            artifact = 'http://127.0.0.1:' + str(artifact_port)
            worker = 'http://127.0.0.1:' + str(worker_port)
            command = [build['server'], '--database=' + str(database), '--addr=127.0.0.1:' + str(api_port), '--artifact-addr=127.0.0.1:' + str(artifact_port), '--worker-addr=127.0.0.1:' + str(worker_port), '--public-url=' + api, '--artifact-origin=' + artifact, '--artifact-directory=' + str(root / 'artifacts'), '--ui=' + build['ui']]
            process = subprocess.Popen(command, stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
            try:
                ready = False
                for _ in range(100):
                    try:
                        with urllib.request.urlopen(api + '/healthz', timeout=1) as response:
                            ready = response.status == 200 and json.load(response)['status'] == 'up'
                    except (OSError, ValueError):
                        pass
                    if ready:
                        break
                    assert process.poll() is None, 'The owned service stopped before startup'
                    time.sleep(0.05)
                assert ready
                summary['checks'].append('Combined service starts with the explicitly selected private worker listener')
                for path in ['/', '/internal/worker/v1/executions/missing/input']:
                    try:
                        response = urllib.request.urlopen(worker + path, timeout=2)
                    except urllib.error.HTTPError as error:
                        assert error.code == 404 and b'<ts-root' not in error.read(4096)
                    else:
                        response.close()
                        raise AssertionError('The private listener exposed UI or unauthenticated work')
                summary['checks'].append('Private listener exposes neither the UI nor unauthenticated attempt data')
            finally:
                if process.poll() is None:
                    process.send_signal(signal.SIGTERM)
                    try:
                        process.wait(timeout=10)
                    except subprocess.TimeoutExpired:
                        os.killpg(process.pid, signal.SIGKILL)
                        process.wait(timeout=3)
                cleanup['processStopped'] = process.poll() is not None
                assert process.returncode == 0, 'The owned service did not stop cleanly'
            connection = sqlite3.connect('file:' + str(database) + '?mode=ro&immutable=1', uri=True)
            assert connection.execute("SELECT count(*) FROM records WHERE bucket IN ('worker-exchanges','runtime-private')").fetchone()[0] == 0
            connection.close()
            summary['checks'].append('Listener startup creates no work ticket or private signing key without a trusted dispatch')
            summary['passed'] = True
    cleanup['temporaryDataRemoved'] = not root.exists()
    (output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    (output / 'cleanup.json').write_text(json.dumps(cleanup, indent=2) + '\n')
    assert all(cleanup.values())
    print(json.dumps(summary, indent=2))


if __name__ == '__main__':
    main()
