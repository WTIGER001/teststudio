import argparse
import json
from pathlib import Path
import subprocess
import uuid


def docker(*arguments, check=True):
    return subprocess.run(['docker', *arguments], capture_output=True, text=True, check=check, timeout=30)


def main():
    parser = argparse.ArgumentParser(description='Check hosted supervisor overlays locally without starting a browser, test suite, or cloud job.')
    parser.add_argument('--go-runtime', type=Path, required=True)
    parser.add_argument('--playwright-runtime', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    summary = {'passed': False, 'cloudOperations': False, 'browserRuns': False, 'checks': [], 'images': []}
    cleanup = {}
    containers = []
    try:
        for path in [args.go_runtime, args.playwright_runtime]:
            build = json.loads(path.read_text())
            image = json.loads(docker('image', 'inspect', build['image']).stdout)[0]
            base = json.loads(docker('image', 'inspect', build['base']).stdout)[0]
            assert image['Id'] == build['image'] and base['Id'] == build['base']
            assert image['Architecture'] == base['Architecture'] == build['architecture']
            assert image['Config']['User'] == '1000:1000'
            assert image['Config']['Entrypoint'] == base['Config']['Entrypoint']
            labels = image['Config']['Labels']
            assert labels['teststudio.job-supervisor'] == 'worker-exchange/v1'
            assert labels['teststudio.discovery-supervisor'] == 'worker-exchange/v1'
            assert labels['teststudio.job-runtime-base'] == base['Id']
            assert labels['teststudio.task-storage'] == 'task-volumes/v1'
            assert set(image['Config']['Volumes']) == {'/tmp', '/output'}
            assert labels['teststudio.job-source-key'] == build['sourceKey']
            for key in ['teststudio.runtime', 'teststudio.runtime-kind', 'teststudio.work-item-bytes', 'teststudio.api-policy']:
                assert labels[key] == base['Config']['Labels'][key]
            name = 'teststudio-job-package-' + uuid.uuid4().hex[:12]
            containers.append(name)
            result = docker('run', '--rm', '--name', name, '--platform=linux/' + build['architecture'], '--read-only', '--network=none', '--cap-drop=ALL', '--security-opt=no-new-privileges', '--entrypoint=/runner/teststudio-job', build['image'], '--runner=' + build['runner'], '--operation=discover', '--exchange-origin=https://jobs.example.test', check=False)
            assert result.returncode == 2 and 'job execution identity is missing or invalid' in result.stderr
            assert not result.stdout.strip()
            name = 'teststudio-job-volumes-' + uuid.uuid4().hex[:12]
            containers.append(name)
            probe = 'test "$(id -u):$(id -g)" = 1000:1000 && test "$(stat -c %a /tmp)" = 1777 && test "$(stat -c %a /output)" = 700 && test "$(stat -c %u:%g /output)" = 1000:1000 && printf temporary > /tmp/probe && printf evidence > /output/probe && ! (printf blocked > /runner/volume-probe) 2>/dev/null && ! (printf blocked > /suite/volume-probe) 2>/dev/null'
            result = docker('run', '--rm', '--name', name, '--platform=linux/' + build['architecture'], '--read-only', '--network=none', '--cap-drop=ALL', '--security-opt=no-new-privileges', '--entrypoint=/bin/sh', build['image'], '-c', probe)
            assert result.returncode == 0 and not result.stdout.strip()
            summary['checks'].append(build['runner'] + ': task volumes initialize non-root ownership, permit scratch writes, and retain a read-only runtime and suite')
            summary['checks'].append(build['runner'] + ': immutable base and runner contracts retained; non-root supervisor executes without ambient work or network')
            summary['images'].append(build)
        summary['passed'] = True
    finally:
        for name in containers:
            docker('rm', '--force', '--volumes', name, check=False)
            state = docker('inspect', name, check=False)
            cleanup[name] = state.returncode != 0 and 'no such' in state.stderr.lower()
        (output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
        (output / 'cleanup.json').write_text(json.dumps(cleanup, indent=2) + '\n')
        assert all(cleanup.values()), 'An owned runtime package check remains'
    print(json.dumps({'passed': summary['passed'], 'checks': len(summary['checks']), 'output': str(output)}))


if __name__ == '__main__':
    main()
