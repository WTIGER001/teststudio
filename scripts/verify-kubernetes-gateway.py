import argparse
import hashlib
import io
import json
from pathlib import Path
import time
import subprocess
import uuid
import zipfile

from lib.local_kubernetes import LocalAPI
from lib.local_kubernetes_cluster import LocalCluster


def main():
    parser = argparse.ArgumentParser(description='Verify new environment-backed suites on the owned local Kubernetes installation.')
    parser.add_argument('--runner', choices=['go', 'playwright', 'all'], default='all')
    parser.add_argument('--retry-preparation-after-fix', action='store_true')
    parser.add_argument('--retry-run-after-fix', action='store_true')
    args = parser.parse_args()
    installation = Path('.data/local-k3s')
    output = Path('output/local-k3s/gateway')
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    setup = json.loads((output / 'installation.json').read_text())
    assert setup['deployed'], 'Deploy the owned gateway profile first'
    cluster = LocalCluster(installation)
    state_file = output / 'references.json'
    state = json.loads(state_file.read_text()) if state_file.exists() else {'project': 'k3s-network', 'suites': {}}
    credential = (installation / 'gateway/reader').read_bytes()
    assert len(credential) >= 32

    def checkpoint():
        state_file.write_text(json.dumps(state, indent=2) + '\n')

    def save(name, value):
        data = json.dumps(value, indent=2).encode()
        assert credential not in data, 'The private fixture credential appeared in public diagnostics'
        (output / name).write_bytes(data + b'\n')

    def capture_worker_logs(runner, run_id):
        for pod in cluster.pods('teststudio-workers'):
            ref = json.loads(pod['metadata'].get('annotations', {}).get('teststudio.io/reference', '{}'))
            if ref.get('runId') != run_id:
                continue
            for container in pod['spec']['containers']:
                command = cluster.command + ['kubectl', '--kubeconfig', '/etc/rancher/k3s/k3s.yaml', 'logs', pod['metadata']['name'], '-n', 'teststudio-workers', '-c', container['name'], '--tail=40', '--limit-bytes=8192']
                result = subprocess.run(command, capture_output=True, timeout=10)
                if result.returncode == 0 and result.stdout:
                    assert credential not in result.stdout, 'The fixture credential appeared in provider logs'
                    (output / (runner + '-' + run_id + '-' + container['name'] + '-provider.log')).write_bytes(result.stdout)

    api = LocalAPI(installation)
    api.login()
    base = '/api/v1/projects/' + state['project']
    try:
        if not state.get('previousRuns'):
            existing = api.call('GET', '/api/v1/projects/k3s-coded/runs')
            state['previousRuns'] = sorted(row['id'] for row in existing['items'])
            assert len(state['previousRuns']) == 12, 'Review existing offline history before changing the fixture'
            checkpoint()
        if not state.get('projectCreated'):
            api.call('POST', '/api/v1/projects', {'id': state['project'], 'name': 'Kubernetes application networking', 'description': 'New independent browser and authenticated API scenarios'})
            state['projectCreated'] = True
            checkpoint()
        if not state.get('applicationCreated'):
            api.call('PUT', base + '/applications/gateway-fixture', {'id': 'gateway-fixture', 'name': 'Gateway demonstration application'}, 0)
            state['applicationCreated'] = True
            checkpoint()
        for runner in (['go', 'playwright'] if args.runner == 'all' else [args.runner]):
            info = state['suites'].setdefault(runner, {'suite': 'go-api-network' if runner == 'go' else 'browser-network', 'uploadKey': str(uuid.uuid4()), 'runKey': str(uuid.uuid4())})
            checkpoint()
            environment = 'api-local' if runner == 'go' else 'browser-local'
            if not info.get('environmentCreated'):
                value = {'id': environment, 'name': 'Private HTTPS API' if runner == 'go' else 'Local browser application', 'applicationId': 'gateway-fixture', 'stage': 'local', 'protected': runner == 'go', 'allowHttp': runner == 'playwright', 'allowPrivateNetwork': True, 'endpoints': {'api': setup['apiOrigin'] if runner == 'go' else setup['browserOrigin']}}
                if runner == 'go':
                    value['credentialBindings'] = {'reader': 'gateway-reader'}
                api.call('PUT', base + '/environments/' + environment, value, 0)
                info['environmentCreated'] = True
                checkpoint()
            suite = base + '/suites/' + info['suite']
            if not info.get('created'):
                api.call('PUT', suite, {'id': info['suite'], 'name': 'Registered application ' + runner, 'applications': ['gateway-fixture'], 'requiresTarget': True}, 0)
                info['created'] = True
                checkpoint()
            if args.retry_preparation_after_fix and info.get('preparation') and not info.get('release'):
                old = api.call('GET', suite + '/source-preparations/' + info['preparation'])
                assert old['data']['state'] == 'failed' and not old['data'].get('cleanupPending'), 'The failed preparation must finish cleanup before a corrected submission'
                save(runner + '-failed-preparation-' + info['preparation'] + '.json', old)
                info.setdefault('failedPreparations', []).append(info.pop('preparation'))
                info['uploadKey'] = str(uuid.uuid4())
                checkpoint()
            if not info.get('preparation'):
                source = Path('tests/kubernetes/environment') / ('go' if runner == 'go' else 'browser')
                buffer = io.BytesIO()
                with zipfile.ZipFile(buffer, 'w', zipfile.ZIP_DEFLATED) as archive:
                    for file in sorted(source.rglob('*')):
                        if file.is_file():
                            archive.writestr(file.relative_to(source).as_posix(), file.read_bytes())
                prepared = api.call('POST', suite + '/source-bundles?name=' + runner + '-network.zip&manifest=teststudio-suite.json', buffer.getvalue(), content_type='application/zip', key=info['uploadKey'])
                info['preparation'] = prepared['id']
                checkpoint()
            until = time.monotonic() + 150
            previous = None
            while True:
                prepared = api.call('GET', suite + '/source-preparations/' + info['preparation'])
                save(runner + '-preparation.json', prepared)
                job = prepared['data']
                phase = (job['state'], job['phase'])
                if phase != previous:
                    print(runner + ' preparation: ' + '/'.join(phase), flush=True)
                    previous = phase
                if job['state'] in ['ready', 'published', 'failed', 'cancelled']:
                    break
                assert time.monotonic() < until, 'Observe the saved preparation; do not create a replacement'
                time.sleep(1)
            assert job['state'] in ['ready', 'published'], 'Inspect the retained preparation diagnostics'
            if not info.get('release'):
                upload = api.call('GET', suite + '/source-uploads/' + job['uploadId'])['upload']
                published = api.call('POST', suite + '/source-uploads/' + upload['id'] + '/publish', {'activate': True}, upload['revision'])
                info['release'] = published['id']
                checkpoint()
            if args.retry_run_after_fix and info.get('run'):
                previous_path = base + '/runs/' + info['run']
                previous = api.call('GET', previous_path)
                assert previous['data']['state'] in ['failed', 'incomplete', 'blocked', 'timed-out'], 'Only a terminal unsuccessful run can receive a corrected execution'
                executions = api.call('GET', previous_path + '/executions')
                assert executions['items'] and all(row['data']['state'] == 'cleaned' for row in executions['items']), 'Finish cleanup of the original execution before a corrected run'
                save(runner + '-unsuccessful-run-' + info['run'] + '.json', previous)
                save(runner + '-unsuccessful-executions-' + info['run'] + '.json', executions)
                for kind in ['artifacts', 'logs']:
                    save(runner + '-unsuccessful-' + kind + '-' + info['run'] + '.json', api.call('GET', previous_path + '/' + kind))
                info.setdefault('unsuccessfulRuns', []).append(info.pop('run'))
                info['runKey'] = str(uuid.uuid4())
                checkpoint()
            if not info.get('run'):
                created = api.call('POST', base + '/runs', {'suiteId': info['suite'], 'environmentId': environment, 'timeoutSeconds': 90}, key=info['runKey'])
                info['run'] = created['id']
                checkpoint()
            path = base + '/runs/' + info['run']
            until = time.monotonic() + 150
            previous = None
            while True:
                record = api.call('GET', path)
                save(runner + '-run.json', record)
                run = record['data']
                capture_worker_logs(runner, info['run'])
                if previous != run['state']:
                    print(runner + ' run: ' + run['state'], flush=True)
                    previous = run['state']
                if run['state'] in ['completed', 'failed', 'cancelled', 'incomplete', 'blocked', 'timed-out']:
                    break
                assert time.monotonic() < until, 'Observe the saved run; do not create a replacement'
                time.sleep(1)
            artifacts = api.call('GET', path + '/artifacts')
            logs = api.call('GET', path + '/logs')
            save(runner + '-artifacts.json', artifacts)
            save(runner + '-logs.json', logs)
            assert run['state'] == 'completed' and run['outcome'] == 'passed', 'Inspect the retained run and logs'
            assert run['evidenceState'] == 'complete' and len(run['results']) == 1
            result = run['results'][0]
            assert result['state'] == 'passed' and result['durationMs'] > 0 and result['executionId']
            expected = 'application/x-ndjson' if runner == 'go' else 'image/png'
            available = [row['data'] for row in artifacts['items'] if row['data']['state'] == 'available']
            assert any(row['contentType'] == expected for row in available)
            for artifact in available:
                grant = api.call('POST', path + '/artifacts/' + artifact['id'] + '/access', {'action': 'download'})
                data = api.download(grant['url'])
                assert len(data) == artifact['size'] and hashlib.sha256(data).hexdigest() == artifact['checksum']
                assert credential not in data, 'The fixture credential appeared in private evidence'
                if artifact['contentType'] == 'image/png':
                    (output / (runner + '-application.png')).write_bytes(data)
                if zipfile.is_zipfile(io.BytesIO(data)):
                    with zipfile.ZipFile(io.BytesIO(data)) as archive:
                        entries = archive.infolist()
                        assert len(entries) <= 10000 and sum(entry.file_size for entry in entries) <= 128 << 20
                        for entry in entries:
                            assert credential not in archive.read(entry), 'The fixture credential appeared inside archived evidence'
            until = time.monotonic() + 70
            while True:
                executions = api.call('GET', path + '/executions')
                save(runner + '-executions.json', executions)
                assert len(executions['items']) == 1
                execution = executions['items'][0]['data']
                if execution['state'] == 'cleaned':
                    break
                assert time.monotonic() < until, 'Observe the existing execution cleanup checkpoint'
                time.sleep(1)
            ref = execution['reference']
            assert ref['profile'] == ('go' if runner == 'go' else 'playwright') + '-kubernetes-environment-v1' and ref['gateway']
            pods = cluster.pods('teststudio-workers')
            assert not any(pod['metadata']['name'] in [ref['container'], ref['gateway']] for pod in pods)
            resources = json.loads(cluster.execute('kubectl', '--kubeconfig', '/etc/rancher/k3s/k3s.yaml', 'get', 'services,networkpolicies', '-n', 'teststudio-workers', '-o', 'json'))
            assert not any(row['metadata']['name'] in [ref['container'], ref['gateway']] for row in resources['items'])
            summary = {'passed': True, 'runner': runner, 'runId': info['run'], 'preparationId': info['preparation'], 'durationMs': result['durationMs'], 'evidenceState': run['evidenceState'], 'artifactsVerified': len(available), 'credentialAbsentFromEvidence': True, 'cleanup': execution['state'], 'browserUsesLocalHTTP': runner == 'playwright', 'apiUsesVerifiedHTTPS': runner == 'go', 'cloudOperations': False}
            save(runner + '-summary.json', summary)
            print(json.dumps(summary), flush=True)
        retained = api.call('GET', '/api/v1/projects/k3s-coded/runs')
        assert sorted(row['id'] for row in retained['items']) == state['previousRuns']
    finally:
        api.logout()


if __name__ == '__main__':
    main()
