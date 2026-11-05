import argparse
import hashlib
import io
import json
from pathlib import Path
import time
import uuid
import zipfile

from lib.local_kubernetes import LocalAPI


def main():
    parser = argparse.ArgumentParser(description='Publish and execute fresh Go and Playwright bundles through the installed local Kubernetes product.')
    parser.add_argument('--runner', choices=['go', 'playwright', 'all'], default='all')
    parser.add_argument('--installation', type=Path, default=Path('.data/local-k3s'))
    parser.add_argument('--output', type=Path, default=Path('output/local-k3s/coded'))
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True, mode=0o700)
    state_file = args.output / 'references.json'
    state = json.loads(state_file.read_text()) if state_file.exists() else {'project': 'k3s-coded', 'suites': {}}

    def checkpoint():
        state_file.write_text(json.dumps(state, indent=2) + '\n')

    api = LocalAPI(args.installation)
    api.login()
    base = '/api/v1/projects/' + state['project']
    try:
        if not state.get('projectCreated'):
            api.call('POST', '/api/v1/projects', {'id': state['project'], 'name': 'Kubernetes coded integration', 'description': 'Fresh independent Go and Playwright worker verification'})
            state['projectCreated'] = True
            checkpoint()
        if not state.get('applicationCreated'):
            api.call('PUT', base + '/applications/workers', {'id': 'workers', 'name': 'Isolated worker examples'}, 0)
            state['applicationCreated'] = True
            checkpoint()
        for runner in (['go', 'playwright'] if args.runner == 'all' else [args.runner]):
            info = state['suites'].setdefault(runner, {'suite': 'go-local' if runner == 'go' else 'browser-local', 'uploadKey': str(uuid.uuid4()), 'runKey': str(uuid.uuid4())})
            checkpoint()
            suite = base + '/suites/' + info['suite']
            if not info.get('created'):
                api.call('PUT', suite, {'id': info['suite'], 'name': 'Local Kubernetes ' + runner, 'applications': ['workers'], 'requiresTarget': False}, 0)
                info['created'] = True
                checkpoint()
            if not info.get('preparation'):
                root = Path('tests/kubernetes/suites') / runner
                buffer = io.BytesIO()
                with zipfile.ZipFile(buffer, 'w', zipfile.ZIP_DEFLATED) as archive:
                    for file in sorted(root.rglob('*')):
                        if file.is_file():
                            archive.writestr(file.relative_to(root).as_posix(), file.read_bytes())
                row = api.call('POST', suite + '/source-bundles?name=' + runner + '-local.zip&manifest=teststudio-suite.json', buffer.getvalue(), content_type='application/zip', key=info['uploadKey'])
                info['preparation'] = row['id']
                checkpoint()
            deadline = time.monotonic() + 120
            last_phase = None
            while True:
                prepared = api.call('GET', suite + '/source-preparations/' + info['preparation'])
                job = prepared['data']
                (args.output / (runner + '-preparation.json')).write_text(json.dumps(prepared, indent=2) + '\n')
                phase = (job['state'], job['phase'])
                if phase != last_phase:
                    print(runner + ' preparation: ' + '/'.join(phase), flush=True)
                    last_phase = phase
                if job['state'] in ['ready', 'published', 'failed', 'cancelled']:
                    break
                assert time.monotonic() < deadline, 'The saved preparation is still active; inspect its checkpoint before continuing'
                time.sleep(1)
            assert job['state'] in ['ready', 'published'], 'Preparation did not succeed: ' + job.get('error', '')
            if not info.get('release'):
                upload = api.call('GET', suite + '/source-uploads/' + job['uploadId'])['upload']
                published = api.call('POST', suite + '/source-uploads/' + upload['id'] + '/publish', {'activate': True}, upload['revision'])
                info['release'] = published['id']
                checkpoint()
            if not info.get('run'):
                created = api.call('POST', base + '/runs', {'suiteId': info['suite'], 'timeoutSeconds': 90}, key=info['runKey'])
                info['run'] = created['id']
                checkpoint()
            run_path = base + '/runs/' + info['run']
            deadline = time.monotonic() + 120
            last_state = None
            while True:
                record = api.call('GET', run_path)
                run = record['data']
                (args.output / (runner + '-run.json')).write_text(json.dumps(record, indent=2) + '\n')
                if run['state'] != last_state:
                    print(runner + ' run: ' + run['state'], flush=True)
                    last_state = run['state']
                if run['state'] in ['completed', 'failed', 'cancelled', 'incomplete']:
                    break
                assert time.monotonic() < deadline, 'The saved run is still active; inspect its checkpoint before continuing'
                time.sleep(1)
            artifacts = api.call('GET', run_path + '/artifacts')
            logs = api.call('GET', run_path + '/logs')
            executions = api.call('GET', run_path + '/executions')
            for kind, value in [('artifacts', artifacts), ('logs', logs), ('executions', executions)]:
                (args.output / (runner + '-' + kind + '.json')).write_text(json.dumps(value, indent=2) + '\n')
            assert run['state'] == 'completed' and run['outcome'] == 'passed', 'The actual coded run did not pass; retained diagnostics are beside its run record'
            assert run['evidenceState'] == 'complete' and len(run['results']) == 1
            result = run['results'][0]
            assert result['state'] == 'passed' and result['durationMs'] > 0 and result['executionId']
            assert logs['segments'] and executions['items']
            available = [row['data'] for row in artifacts['items'] if row['data']['state'] == 'available']
            if runner == 'playwright':
                available = [item for item in available if item['contentType'] == 'image/png']
            assert available, 'The coded run retained no expected downloadable evidence'
            artifact = available[0]
            grant = api.call('POST', run_path + '/artifacts/' + artifact['id'] + '/access', {'action': 'download'})
            body = api.download(grant['url'])
            assert len(body) == artifact['size'] and hashlib.sha256(body).hexdigest() == artifact['checksum']
            (args.output / (runner + '-private-evidence')).write_bytes(body)
            summary = {'passed': True, 'runner': runner, 'preparationId': info['preparation'], 'runId': info['run'], 'durationMs': result['durationMs'], 'evidenceState': run['evidenceState'], 'artifactCount': len(artifacts['items']), 'logSegments': len(logs['segments']), 'privateDownloadVerified': True, 'hostBrowserRuns': False, 'cloudOperations': False}
            (args.output / (runner + '-summary.json')).write_text(json.dumps(summary, indent=2) + '\n')
            print(json.dumps(summary), flush=True)
    finally:
        api.logout()


if __name__ == '__main__':
    main()
