import argparse
import hashlib
import http.client
import http.cookiejar
import json
from pathlib import Path
import socket
import ssl
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid


def main():
    parser = argparse.ArgumentParser(description='Verify normal OIDC, catalog, HTTPS execution, and private evidence against the dedicated local Kubernetes installation.')
    parser.add_argument('--installation', type=Path, default=Path('.data/local-k3s'))
    parser.add_argument('--output', type=Path, default=Path('output/local-k3s/probe'))
    parser.add_argument('--port', type=int, default=18443)
    args = parser.parse_args()
    ownership = json.loads((args.installation / 'ownership.json').read_text())
    assert ownership['cluster'] == 'teststudio-local' and ownership['profile'] == 'teststudio-k3s'
    args.output.mkdir(parents=True, exist_ok=True, mode=0o700)
    origin = 'https://studio.teststudio.test'
    context = ssl.create_default_context(cafile=str(args.installation / 'tls/server.crt'))
    context.load_verify_locations(cafile=str(args.installation / 'fixture-tls/ca.crt'))
    allowed = {'studio.teststudio.test', 'identity.teststudio.test', 'evidence.teststudio.test'}

    class LocalConnection(http.client.HTTPSConnection):
        def connect(self):
            assert self.host in allowed and self.port == 443, 'Only the owned local installation is allowed'
            connection = socket.create_connection(('127.0.0.1', args.port), timeout=self.timeout)
            try:
                self.sock = context.wrap_socket(connection, server_hostname=self.host)
            except BaseException:
                connection.close()
                raise

    class LocalTLS(urllib.request.HTTPSHandler):
        def https_open(self, request):
            return self.do_open(LocalConnection, request, context=context)

    browser = urllib.request.build_opener(urllib.request.ProxyHandler({}), LocalTLS(), urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
    with browser.open(origin + '/auth/login/local-k3s?returnTo=/projects', timeout=15) as response:
        assert response.status == 200 and response.url == origin + '/projects', 'OIDC did not return to the application'
    csrf = ''

    def call(method, path, body=None, revision=None):
        headers = {'Origin': origin, 'X-CSRF-Token': csrf, 'Idempotency-Key': str(uuid.uuid4())}
        if revision is not None:
            headers['If-Match'] = '"' + str(revision) + '"'
        payload = None if body is None else json.dumps(body).encode()
        if payload is not None:
            headers['Content-Type'] = 'application/json'
        request = urllib.request.Request(origin + path, data=payload, headers=headers, method=method)
        try:
            with browser.open(request, timeout=15) as response:
                return json.load(response)
        except urllib.error.HTTPError as error:
            raise RuntimeError(method + ' ' + path + ': ' + str(error.code) + ' ' + error.read(4096).decode()) from None

    session = call('GET', '/auth/session')
    assert session['user'] and session['session']['provider'] == 'local-k3s', 'A normal provider session is required'
    csrf = session['session']['csrfToken']
    try:
        project = 'k3s-integration'
        base = '/api/v1/projects/' + project
        checkpoint = args.output / 'run-reference.json'
        if checkpoint.exists():
            run_id = json.loads(checkpoint.read_text())['runId']
        else:
            call('POST', '/api/v1/projects', {'id': project, 'name': 'Local Kubernetes integration', 'description': 'New independently authored Kubernetes acceptance probe'})
            call('PUT', base + '/applications/probe', {'id': 'probe', 'name': 'Local TLS fixture'}, 0)
            call('PUT', base + '/environments/local', {'id': 'local', 'name': 'Dedicated k3s fixture', 'applicationId': 'probe', 'stage': 'local', 'endpoints': {'api': 'https://probe.teststudio.test'}}, 0)
            call('PUT', base + '/suites/availability', {'id': 'availability', 'name': 'Kubernetes availability', 'applications': ['probe'], 'requiresTarget': True}, 0)
            call('POST', base + '/suites/availability/tests', {'id': 'K3S-HTTPS-001', 'title': 'Read the dedicated Kubernetes TLS endpoint', 'category': 'Availability', 'applicationId': 'probe', 'type': 'https', 'runner': 'http', 'state': 'enabled', 'safety': 'safe', 'rationale': 'Reads only the newly authored local fixture', 'tags': ['local-kubernetes'], 'dependencies': [], 'requiredBindings': [], 'timeoutSeconds': 10, 'check': {'endpoint': 'api', 'path': '/health', 'method': 'GET', 'statusCodes': [200], 'headers': {'X-TestStudio-Fixture': 'local-k3s'}, 'json': {'status': 'ready', 'source': 'new-local-kubernetes-fixture'}, 'minCertificateDays': 1}})
            draft = call('POST', base + '/suites/availability/catalog-drafts', {})
            call('POST', base + '/suites/availability/catalog-drafts/' + draft['id'] + '/publish', {'activate': True}, draft['revision'])
            created = call('POST', base + '/runs', {'suiteId': 'availability', 'environmentId': 'local', 'timeoutSeconds': 60})
            run_id = created['id']
            checkpoint.write_text(json.dumps({'projectId': project, 'runId': run_id}, indent=2) + '\n')
        run_path = base + '/runs/' + run_id
        listed = call('GET', base + '/runs')
        assert len(listed['items']) == 1 and listed['items'][0]['id'] == run_id, 'The isolated probe project has unexpected runs'
        deadline = time.monotonic() + 30
        while True:
            record = call('GET', run_path)
            run = record['data']
            if run['state'] in ['completed', 'cancelled', 'failed']:
                break
            assert time.monotonic() < deadline, 'The existing run is still pending; inspect its checkpoint before continuing'
            time.sleep(0.25)
        (args.output / 'run.json').write_text(json.dumps(record, indent=2) + '\n')
        assert run['state'] == 'completed' and run['outcome'] == 'passed', 'The actual probe did not pass'
        assert run['evidenceState'] == 'complete', 'The actual probe evidence is incomplete'
        assert len(run['results']) == 1
        result = run['results'][0]
        observation = result['http']
        assert result['testId'] == 'K3S-HTTPS-001' and result['state'] == 'passed' and result['durationMs'] > 0
        assert observation['status'] == 200 and observation['certificateExpires'] and observation['trustBundle']['id'] == 'local-k3s-probe'
        assert observation['timingsMs']['tls'] > 0
        evidence = {}
        for kind in ['events', 'attempts', 'artifacts', 'logs']:
            page = call('GET', run_path + '/' + kind)
            (args.output / (kind + '.json')).write_text(json.dumps(page, indent=2) + '\n')
            entries = page['segments' if kind == 'logs' else 'items']
            evidence[kind] = len(entries)
            assert entries, kind + ' was not retained'
        artifacts = json.loads((args.output / 'artifacts.json').read_text())['items']
        artifact = next(item['data'] for item in artifacts if item['data']['state'] == 'available')
        grant = call('POST', run_path + '/artifacts/' + artifact['id'] + '/access', {'action': 'download'})
        assert urllib.parse.urlsplit(grant['url']).hostname == 'evidence.teststudio.test'
        with browser.open(grant['url'], timeout=15) as response:
            body = response.read()
        assert len(body) == artifact['size'] and hashlib.sha256(body).hexdigest() == artifact['checksum']
        (args.output / 'private-evidence.txt').write_bytes(body)
        summary = {'passed': True, 'runId': run_id, 'projectId': project, 'authentication': 'OIDC code flow with PKCE and validated private TLS', 'httpStatus': observation['status'], 'durationMs': result['durationMs'], 'tlsMs': observation['timingsMs']['tls'], 'certificateExpires': observation['certificateExpires'], 'evidence': evidence, 'privateDownloadVerified': True, 'browserRuns': False, 'cloudOperations': False}
        (args.output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    finally:
        assert call('POST', '/auth/logout', {})['signedOut'], 'The owned HTTP session was not revoked'
    print(json.dumps(summary))


if __name__ == '__main__':
    main()
