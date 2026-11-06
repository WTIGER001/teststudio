import argparse
import hashlib
from html.parser import HTMLParser
import http.cookiejar
import json
from pathlib import Path
import subprocess
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid


class LoginForm(HTMLParser):
    def __init__(self):
        super().__init__()
        self.forms = []
        self.current = None

    def handle_starttag(self, tag, attributes):
        values = dict(attributes)
        if tag == 'form':
            self.current = {'action': values.get('action', ''), 'fields': {}}
            self.forms.append(self.current)
        if tag == 'input' and self.current is not None and values.get('name'):
            self.current['fields'][values['name']] = values.get('value', '')

    def handle_endtag(self, tag):
        if tag == 'form':
            self.current = None


class CoreAPI:
    def __init__(self, origin, identity_host):
        self.origin = origin
        self.identity_host = identity_host
        self.csrf = ''
        self.browser = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

    def login(self, credential):
        with self.browser.open(self.origin + '/auth/login/cognito?returnTo=/projects', timeout=30) as response:
            location = response.url
            assert urllib.parse.urlsplit(location).hostname == self.identity_host, 'Unexpected identity host'
            parser = LoginForm()
            parser.feed(response.read(1000000).decode())
        form = next(item for item in parser.forms if 'username' in item['fields'] and 'password' in item['fields'])
        target = urllib.parse.urljoin(location, form['action'])
        parsed = urllib.parse.urlsplit(target)
        assert parsed.scheme == 'https' and parsed.hostname == self.identity_host, 'Unexpected credential destination'
        form['fields'].update(username=credential['username'], password=credential['password'])
        request = urllib.request.Request(target, data=urllib.parse.urlencode(form['fields']).encode(), headers={'Content-Type': 'application/x-www-form-urlencoded', 'Origin': 'https://' + self.identity_host, 'Referer': location})
        with self.browser.open(request, timeout=30) as response:
            assert response.url == self.origin + '/projects', 'Normal OIDC callback did not complete'
        session = self.call('GET', '/auth/session')
        if session.get('session'):
            self.csrf = session['session']['csrfToken']
        assert session['user'] and session['session']['provider'] == 'cognito', 'A normal OIDC session is required'
        return session['user']

    def call(self, method, path, value=None, revision=None, key=None):
        assert path.startswith('/') and not path.startswith('//')
        headers = {'Accept': 'application/json', 'Origin': self.origin, 'X-CSRF-Token': self.csrf}
        if revision is not None:
            headers['If-Match'] = '"' + str(revision) + '"'
        headers['Idempotency-Key'] = key or str(uuid.uuid4())
        payload = None if value is None else json.dumps(value).encode()
        if payload is not None:
            headers['Content-Type'] = 'application/json'
        request = urllib.request.Request(self.origin + path, data=payload, method=method, headers=headers)
        try:
            with self.browser.open(request, timeout=30) as response:
                return json.load(response)
        except urllib.error.HTTPError as error:
            raise RuntimeError(method + ' ' + path + ': HTTP ' + str(error.code)) from None


def main():
    parser = argparse.ArgumentParser(description='Inspect one deployed AWS core workflow. Reuse its run checkpoint; do not repeat execution.')
    for name in ['profile', 'region', 'account', 'stack', 'credential-secret', 'output']:
        parser.add_argument('--' + name, required=True)
    args = parser.parse_args()
    output = Path(args.output)
    output.mkdir(parents=True, exist_ok=True)
    command = ['aws', '--profile', args.profile, '--region', args.region, '--no-cli-pager']

    def aws(arguments):
        result = subprocess.run(command + arguments, capture_output=True, text=True)
        if result.returncode:
            raise RuntimeError('AWS ' + ' '.join(arguments[:2]) + ' failed')
        return json.loads(result.stdout)

    assert aws(['sts', 'get-caller-identity'])['Account'] == args.account, 'Unexpected AWS account'
    stack = aws(['cloudformation', 'describe-stacks', '--stack-name', args.stack])['Stacks'][0]
    assert stack['StackStatus'] in ['CREATE_COMPLETE', 'UPDATE_COMPLETE'], 'The existing stack is not ready'
    assert any(tag == {'Key': 'Application', 'Value': 'TestStudio'} for tag in stack.get('Tags', [])), 'Installation ownership is required'
    outputs = {item['OutputKey']: item['OutputValue'] for item in stack['Outputs']}
    origin = outputs['UiUrl']
    evidence_origin = outputs['ArtifactUrl']
    parameters = {item['ParameterKey']: item['ParameterValue'] for item in stack['Parameters']}
    issuer = parameters['BootstrapIssuer']
    pool = issuer.rsplit('/', 1)[1]
    identity = aws(['cognito-idp', 'describe-user-pool', '--user-pool-id', pool])['UserPool']
    identity_host = identity['Domain'] + '.auth.' + args.region + '.amazoncognito.com'
    credentials = json.loads(aws(['secretsmanager', 'get-secret-value', '--secret-id', args.credential_secret])['SecretString'])
    api = CoreAPI(origin, identity_host)
    try:
        with urllib.request.urlopen(origin + '/healthz', timeout=30) as response:
            assert response.status == 200
        modes = api.call('GET', '/auth/providers')
        assert not modes['localEnabled'], 'Loopback administrator sign-in must remain unavailable'
        assert any(item['id'] == 'cognito' for item in modes['providers']), 'The dedicated provider is missing'
        api.login(credentials)
        credentials = None
        project = 'aws-core'
        base = '/api/v1/projects/' + project
        description = 'Independently authored TestStudio AWS core deployment check'
        projects = api.call('GET', '/api/v1/projects')['items']
        existing = next((item for item in projects if item['id'] == project), None)
        if existing is None:
            api.call('POST', '/api/v1/projects', {'id': project, 'name': 'AWS deployment check', 'description': description})
        else:
            assert existing['data']['description'] == description, 'The existing project is not owned by this check'

        def ensure(kind, identifier, value):
            current = next((item for item in api.call('GET', base + '/' + kind)['items'] if item['id'] == identifier), None)
            return current or api.call('PUT', base + '/' + kind + '/' + identifier, {'id': identifier, **value}, 0)

        ensure('applications', 'teststudio', {'name': 'TestStudio hosted core'})
        ensure('environments', 'mmdev', {'name': 'MMDev ECS deployment', 'applicationId': 'teststudio', 'stage': 'development', 'endpoints': {'api': origin}})
        suite = ensure('suites', 'availability', {'name': 'Hosted availability', 'applications': ['teststudio'], 'requiresTarget': True})
        if not suite['data'].get('activeReleaseId'):
            tests = api.call('GET', base + '/suites/availability/tests')['items']
            if not any(item['id'] == 'AWS-CORE-HTTPS-001' for item in tests):
                api.call('POST', base + '/suites/availability/tests', {'id': 'AWS-CORE-HTTPS-001', 'title': 'Hosted TestStudio health responds over verified HTTPS', 'applicationId': 'teststudio', 'category': 'Availability', 'type': 'https', 'runner': 'http', 'state': 'enabled', 'safety': 'safe', 'rationale': 'Reads only this new TestStudio installation health endpoint', 'tags': ['aws-deployment'], 'dependencies': [], 'requiredBindings': [], 'timeoutSeconds': 15, 'check': {'endpoint': 'api', 'path': '/healthz', 'method': 'GET', 'statusCodes': [200], 'minCertificateDays': 1}})
            drafts = api.call('GET', base + '/suites/availability/catalog-drafts')['items']
            draft = next((item for item in drafts if not item['data'].get('publishedReleaseId') and [test['id'] for test in item['data']['tests']] == ['AWS-CORE-HTTPS-001']), None)
            if draft is None:
                draft = api.call('POST', base + '/suites/availability/catalog-drafts', {})
            api.call('POST', base + '/suites/availability/catalog-drafts/' + draft['id'] + '/publish', {'activate': True}, draft['revision'], key='aws-core-publish-' + draft['id'])
        reference = output / 'run-reference.json'
        if reference.exists():
            saved = json.loads(reference.read_text())
            assert saved['stackId'] == stack['StackId'], 'The checkpoint belongs to another installation'
            run_id = saved['runId']
        else:
            key = 'aws-core-' + hashlib.sha256(stack['StackId'].encode()).hexdigest()[:32]
            created = api.call('POST', base + '/runs', {'suiteId': 'availability', 'environmentId': 'mmdev', 'executionLocation': 'ecs', 'timeoutSeconds': 60}, key=key)
            run_id = created['id']
            reference.write_text(json.dumps({'stackId': stack['StackId'], 'projectId': project, 'runId': run_id}, indent=2) + '\n')
        run_path = base + '/runs/' + run_id
        deadline = time.monotonic() + 75
        while True:
            record = api.call('GET', run_path)
            run = record['data']
            if run['state'] in ['completed', 'cancelled', 'timed-out', 'blocked', 'incomplete']:
                break
            assert time.monotonic() < deadline, 'The original run remains pending; inspect its checkpoint before continuing'
            time.sleep(0.5)
        (output / 'run.json').write_text(json.dumps(record, indent=2) + '\n')
        assert run['state'] == 'completed' and run['outcome'] == 'passed', 'The original deployment probe did not pass'
        assert run['evidenceState'] == 'complete', 'Probe evidence is incomplete'
        assert len(run['results']) == 1 and run['results'][0]['durationMs'] > 0
        evidence = {}
        for kind in ['attempts', 'events', 'logs', 'artifacts']:
            page = api.call('GET', run_path + '/' + kind)
            (output / (kind + '.json')).write_text(json.dumps(page, indent=2) + '\n')
            entries = page['segments' if kind == 'logs' else 'items']
            assert entries, kind + ' was not retained'
            evidence[kind] = len(entries)
        artifacts = json.loads((output / 'artifacts.json').read_text())['items']
        artifact = next(item['data'] for item in artifacts if item['data']['state'] == 'available')
        grant = api.call('POST', run_path + '/artifacts/' + artifact['id'] + '/access', {'action': 'download'})
        assert grant['url'].startswith(evidence_origin + '/'), 'Unexpected evidence origin'
        with urllib.request.urlopen(grant['url'], timeout=30) as response:
            body = response.read(1000000)
        assert len(body) == artifact['size'] and hashlib.sha256(body).hexdigest() == artifact['checksum']
        summary = {'passed': True, 'origin': origin, 'runId': run_id, 'projectId': project, 'authentication': 'Actual Cognito OIDC authorization-code flow and application session', 'durationMs': run['results'][0]['durationMs'], 'httpStatus': run['results'][0]['http']['status'], 'tlsMs': run['results'][0]['http']['timingsMs']['tls'], 'evidence': evidence, 'privateDownloadVerified': True, 'scope': 'One native HTTPS probe on the deployed ECS core; coded workers and complete provider acceptance remain unverified'}
        (output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
        print(json.dumps(summary))
    finally:
        if api.csrf:
            assert api.call('POST', '/auth/logout', {})['signedOut'], 'The owned application session was not revoked'


if __name__ == '__main__':
    try:
        main()
    except urllib.error.URLError:
        raise SystemExit('The deployment HTTP check failed; no credential or redirect URL is printed.') from None
