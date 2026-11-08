import json
import os
from pathlib import Path
import pwd
import subprocess
import sys
import urllib.parse


def command(arguments, **options):
    result = subprocess.run(arguments, capture_output=True, text=True, **options)
    if result.returncode:
        raise RuntimeError('Recording installation command failed: ' + arguments[0])
    return result.stdout


def install():
    source = Path(__file__).resolve().parent
    manifest = json.loads((source / 'manifest.json').read_text())
    region = manifest['region']

    def secret(reference):
        return json.loads(command(['aws', '--region', region, 'secretsmanager', 'get-secret-value', '--secret-id', reference]))['SecretString']

    identity = json.loads(secret(manifest['serverSecret']))
    database = urllib.parse.urlsplit(secret(manifest['databaseSecret']))
    query = dict(urllib.parse.parse_qsl(database.query))
    if database.scheme not in ['postgres', 'postgresql'] or query.get('sslmode') != 'verify-full':
        raise RuntimeError('The recording worker requires the shared verified PostgreSQL connection')
    query['sslrootcert'] = '/opt/teststudio-recording/rds-ca-bundle.pem'
    database_url = urllib.parse.urlunsplit(database._replace(query=urllib.parse.urlencode(query)))
    user = pwd.getpwnam('teststudio-recording')
    protected = Path('/etc/teststudio-recording')
    protected.chmod(0o700)
    os.chown(protected, user.pw_uid, user.pw_gid)

    def private(name, data):
        target = protected / name
        target.write_text(data)
        target.chmod(0o600)
        os.chown(target, user.pw_uid, user.pw_gid)

    private('server.crt', identity['certificate'])
    private('server.key', identity['key'])
    private('client-ca.crt', identity['clientCA'])
    private('database', database_url)
    private('settings.json', json.dumps(manifest))
    root = Path('/opt/teststudio-recording')
    for name in ['teststudio-recording-worker', 'start.py', 'seccomp.json', 'rds-ca-bundle.pem']:
        data = (source / name).read_bytes()
        target = root / name
        target.write_bytes(data)
        target.chmod(0o755 if name == 'teststudio-recording-worker' else 0o644)
    registry = manifest['runtimeImage'].split('/')[0]
    login = command(['aws', '--region', region, 'ecr', 'get-login-password'])
    private_docker = '/run/teststudio-recording-docker'
    os.makedirs(private_docker, mode=0o700, exist_ok=True)
    try:
        command(['docker', '--config', private_docker, 'login', '--username', 'AWS', '--password-stdin', registry], input=login)
        login = None
        for image in [manifest['runtimeImage'], manifest['gatewayImage']]:
            command(['docker', '--config', private_docker, 'pull', image])
    finally:
        import shutil
        shutil.rmtree(private_docker)
    unit = '''[Unit]
Description=Private TestStudio recording worker
After=network-online.target docker.service
Requires=docker.service
[Service]
Type=simple
User=teststudio-recording
Group=teststudio-recording
SupplementaryGroups=docker
ExecStart=/usr/bin/python3 /opt/teststudio-recording/start.py
Restart=on-failure
RestartSec=95
TimeoutStopSec=100
UMask=0077
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/lib/teststudio-recording
StandardOutput=null
StandardError=journal
[Install]
WantedBy=multi-user.target
'''
    Path('/etc/systemd/system/teststudio-recording.service').write_text(unit)
    command(['systemctl', 'daemon-reload'])
    command(['systemctl', 'enable', '--now', 'teststudio-recording.service'])
    print(json.dumps({'installed': True, 'sourceCommit': manifest['sourceCommit'], 'capacity': manifest['capacity']}))


if __name__ == '__main__':
    try:
        install()
    except Exception:
        sys.exit('Private recording worker installation failed; inspect sanitized service status.')
