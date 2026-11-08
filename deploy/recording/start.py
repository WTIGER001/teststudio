import json
import os
from pathlib import Path


root = Path('/etc/teststudio-recording')
settings = json.loads((root / 'settings.json').read_text())
os.environ['TESTSTUDIO_DATABASE'] = (root / 'database').read_text()
os.environ['HOME'] = '/var/lib/teststudio-recording'
arguments = [
    '/opt/teststudio-recording/teststudio-recording-worker',
    '--addr=0.0.0.0:8045', '--worker-id=recording',
    '--database-env=TESTSTUDIO_DATABASE',
    '--tls-certificate=' + str(root / 'server.crt'),
    '--tls-key=' + str(root / 'server.key'),
    '--client-ca=' + str(root / 'client-ca.crt'),
    '--runtime-image=' + settings['runtimeImage'],
    '--gateway-image=' + settings['gatewayImage'],
    '--seccomp=/opt/teststudio-recording/seccomp.json',
    '--origin-policy=' + str(root / 'origins.json'),
    '--directory=/var/lib/teststudio-recording',
    '--capacity=' + str(settings['capacity']), '--idle-limit=2m',
]
os.execv(arguments[0], arguments)
