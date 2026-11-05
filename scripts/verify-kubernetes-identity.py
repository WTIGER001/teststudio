import hashlib
import json
from pathlib import Path
import urllib.error

from lib.local_kubernetes import LocalAPI


def main():
    root = Path('output/local-k3s/identity')
    root.mkdir(parents=True, exist_ok=True, mode=0o700)
    reference = root / 'references.json'
    state = json.loads(reference.read_text()) if reference.exists() else {}
    provider = 'local-k3s-review'
    issuer = 'https://identity-two.teststudio.test/review'
    project = '/api/v1/projects/k3s-coded'
    group = 'k3s-provider-reviewers'
    assignment_path = project + '/role-assignments/group/' + group + '/builtin.teststudio.viewer'
    assignment_body = {'principalKind': 'group', 'principalId': group, 'roleId': 'builtin.teststudio.viewer', 'reason': 'Authorize the local provider reviewer for this project only'}

    def save(name, data):
        (root / (name + '.json')).write_text(json.dumps(data, indent=2) + '\n')

    def checkpoint():
        save('references', state)

    def optional(api, path):
        try:
            return api.call('GET', path)
        except RuntimeError as error:
            if ': 404 ' not in str(error):
                raise
            return None

    def denied(api, method, path, body=None):
        try:
            api.call(method, path, body)
        except RuntimeError as error:
            assert ': 403 ' in str(error), 'Expected an explicit project permission denial'
            return
        raise AssertionError('The provider reviewer unexpectedly received privileged access')

    admin = LocalAPI(Path('.data/local-k3s'))
    first = admin.login()
    state.setdefault('administratorUserId', first['user']['id'])
    assert state['administratorUserId'] == first['user']['id']
    try:
        stored = optional(admin, '/api/v1/identity-providers/' + provider)
        if not stored:
            stored = admin.call('POST', '/api/v1/identity-providers', {'id': provider, 'name': 'Local Kubernetes review identity', 'issuer': issuer, 'clientId': 'teststudio-local-review', 'disabled': False, 'autoProvision': True, 'scopes': ['openid', 'profile', 'email'], 'groupClaim': 'groups', 'groupMappings': {}, 'membershipTtlSeconds': 1800, 'reason': 'Register a second independent issuer for local Kubernetes acceptance'})
        assert stored['data']['issuer'] == issuer and stored['data']['callbackUrl'] == 'https://studio.teststudio.test/auth/callback/' + provider
        state['providerRegistered'] = True
        checkpoint()
        if not state.get('initialIsolationVerified'):
            reviewer = LocalAPI(Path('.data/local-k3s'))
            session = reviewer.login(provider)
            try:
                assert session['user']['id'] != first['user']['id']
                assert session['user']['email'] == json.loads(Path('tests/kubernetes/fixture/secondary-identity.json').read_text())['email']
                state['reviewerUserId'] = session['user']['id']
                denied(reviewer, 'GET', project)
                denied(reviewer, 'GET', '/api/v1/projects/k3s-integration')
                denied(reviewer, 'GET', '/api/v1/users')
                own = reviewer.call('GET', '/api/v1/me')
                save('initial-profile', own)
                state['initialIsolationVerified'] = True
                checkpoint()
            finally:
                reviewer.logout()
        expected_group = {'id': group, 'name': 'Local provider reviewers', 'projectId': 'k3s-coded', 'managedBy': 'oidc:' + provider}
        stored_group = optional(admin, '/api/v1/groups/' + group)
        if stored_group is None:
            stored_group = admin.call('POST', '/api/v1/groups', expected_group)
        assert stored_group['data']['projectId'] == 'k3s-coded' and stored_group['data']['managedBy'] == 'oidc:' + provider
        if not optional(admin, assignment_path):
            admin.call('POST', project + '/role-assignments', assignment_body)
        stored = admin.call('GET', '/api/v1/identity-providers/' + provider)
        if stored['data']['groupMappings'] != {'reviewers': group}:
            definition = {key: stored['data'][key] for key in ['id', 'name', 'issuer', 'clientId', 'scopes', 'disabled', 'autoProvision', 'groupClaim', 'groupMappings', 'membershipTtlSeconds']}
            definition.update(groupClaim='groups', groupMappings={'reviewers': group}, reason='Map the second issuer to this project-owned reviewer group')
            stored = admin.call('PUT', '/api/v1/identity-providers/' + provider, definition, stored['revision'])
        state['providerRevision'] = stored['revision']
        checkpoint()
        reviewer = LocalAPI(Path('.data/local-k3s'))
        session = reviewer.login(provider)
        try:
            assert session['user']['id'] == state['reviewerUserId'] and session['user']['id'] != first['user']['id']
            reviewer.call('GET', project)
            denied(reviewer, 'GET', '/api/v1/projects/k3s-integration')
            denied(reviewer, 'GET', '/api/v1/identity-providers')
            denied(reviewer, 'GET', '/api/v1/users')
            links = {}
            for label, user in [('administrator', first['user']['id']), ('reviewer', session['user']['id'])]:
                rows = admin.call('GET', '/api/v1/users/' + user + '/identities')['items']
                assert len(rows) == 1
                links[label] = rows[0]['data']
            assert links['administrator']['subject'] == links['reviewer']['subject'] and links['administrator']['issuer'] != links['reviewer']['issuer']
            assert links['reviewer']['issuer'] == issuer
            save('issuer-links', links)
            save('mapped-profile', reviewer.call('GET', '/api/v1/me'))
            old = json.loads(Path('output/local-k3s/coded/references.json').read_text())['suites']['playwright']['run']
            path = project + '/runs/' + old
            reviewer.call('GET', path)
            artifacts = reviewer.call('GET', path + '/artifacts')['items']
            artifact = next(row['data'] for row in artifacts if row['data']['contentType'] == 'image/png' and row['data']['state'] == 'available')
            grant = reviewer.call('POST', path + '/artifacts/' + artifact['id'] + '/access', {'action': 'download'})
            body = reviewer.download(grant['url'])
            assert len(body) == artifact['size'] and hashlib.sha256(body).hexdigest() == artifact['checksum']
            (root / 'reviewer-private-evidence.png').write_bytes(body)
            state['artifact'] = {'runId': old, 'artifactId': artifact['id'], 'checksum': artifact['checksum']}
            if not state.get('revocationVerified'):
                assignment = admin.call('GET', assignment_path)
                admin.call('DELETE', assignment_path, {'reason': 'Verify immediate withdrawal of project evidence access'}, assignment['revision'])
                state['assignmentRevoked'] = True
                checkpoint()
                denied(reviewer, 'GET', project)
                denied(reviewer, 'GET', path)
                denied(reviewer, 'POST', path + '/artifacts/' + artifact['id'] + '/access', {'action': 'download'})
                try:
                    with reviewer.browser.open(grant['url'], timeout=15):
                        raise AssertionError('An existing artifact grant survived project access revocation')
                except urllib.error.HTTPError as error:
                    assert error.code == 403, 'The old evidence grant must reject revoked access'
                state['revocationVerified'] = True
                checkpoint()
            if not optional(admin, assignment_path):
                admin.call('POST', project + '/role-assignments', assignment_body)
            state['assignmentRestored'] = True
            checkpoint()
            reviewer.call('GET', path)
            admin.call('GET', '/api/v1/projects/k3s-integration')
            summary = {'passed': True, 'providers': ['local-k3s', provider], 'issuerSeparationVerified': True, 'matchingEmailAndSubjectDoNotMerge': True, 'normalOIDCSignIn': True, 'projectOwnedGroupMapping': True, 'crossProjectDenied': True, 'installationAdministrationDenied': True, 'privateReportAccessVerified': True, 'privateEvidenceChecksumVerified': True, 'existingSessionAccessWithdrawalVerified': state['revocationVerified'], 'existingGrantDeniedDuringRevocation': True, 'reviewerRoleRestored': state['assignmentRestored'], 'testRunsCreated': 0, 'cloudOperations': False, 'hostBrowserStarted': False}
            save('summary', summary)
            print(json.dumps(summary), flush=True)
        finally:
            reviewer.logout()
    finally:
        admin.logout()


if __name__ == '__main__':
    main()
