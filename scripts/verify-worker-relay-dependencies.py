import json
import os
from pathlib import Path
import subprocess

root = Path(__file__).resolve().parents[1]
expected = {
    'github.com/wtiger001/teststudio/internal/identifier',
    'github.com/wtiger001/teststudio/internal/workerwire',
    'github.com/wtiger001/teststudio/adapters/workerexchange/relay',
    'github.com/wtiger001/teststudio/cmd/teststudio-exchange',
}
result = subprocess.run(
    ['go', 'list', '-deps', './cmd/teststudio-exchange'],
    cwd=root,
    env={**os.environ, 'GOWORK': 'off', 'GOFLAGS': ''},
    capture_output=True,
    text=True,
    timeout=30,
)
if result.returncode:
    raise SystemExit('Cannot inspect worker relay production dependencies.')
packages = result.stdout.splitlines()
internal = {p for p in packages if p.startswith('github.com/wtiger001/teststudio/')}
external = [p for p in packages if '.' in p.split('/')[0] and p not in internal]
if internal != expected or external:
    raise SystemExit('Worker relay dependencies exceed its standard-library and wire-helper boundary.')
print(json.dumps({'status': 'passed', 'productionPackages': sorted(internal), 'externalPackages': external}, indent=2))
