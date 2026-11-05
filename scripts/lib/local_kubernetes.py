import http.client
import http.cookiejar
import json
import socket
import ssl
import urllib.error
import urllib.parse
import urllib.request
import uuid


class LocalAPI:
    def __init__(self, installation, port=18443):
        ownership = json.loads((installation / 'ownership.json').read_text())
        assert ownership['cluster'] == 'teststudio-local' and ownership['profile'] == 'teststudio-k3s'
        self.origin = 'https://studio.teststudio.test'
        self.csrf = ''
        context = ssl.create_default_context(cafile=str(installation / 'tls/server.crt'))
        context.load_verify_locations(cafile=str(installation / 'fixture-tls/ca.crt'))
        allowed = {'studio.teststudio.test', 'identity.teststudio.test', 'identity-two.teststudio.test', 'evidence.teststudio.test'}

        class LocalConnection(http.client.HTTPSConnection):
            def connect(self):
                assert self.host in allowed and self.port == 443, 'Only the owned local installation is allowed'
                connection = socket.create_connection(('127.0.0.1', port), timeout=self.timeout)
                try:
                    self.sock = context.wrap_socket(connection, server_hostname=self.host)
                except BaseException:
                    connection.close()
                    raise

        class LocalTLS(urllib.request.HTTPSHandler):
            def https_open(self, request):
                return self.do_open(LocalConnection, request, context=context)

        self.browser = urllib.request.build_opener(urllib.request.ProxyHandler({}), LocalTLS(), urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

    def login(self, provider='local-k3s'):
        assert provider in {'local-k3s', 'local-k3s-review'}
        with self.browser.open(self.origin + '/auth/login/' + provider + '?returnTo=/projects', timeout=15) as response:
            assert response.status == 200 and response.url == self.origin + '/projects', 'OIDC did not return to the application'
        session = self.call('GET', '/auth/session')
        assert session['user'] and session['session']['provider'] == provider, 'A normal provider session is required'
        self.csrf = session['session']['csrfToken']
        return session

    def call(self, method, path, body=None, revision=None, content_type=None, key=None):
        assert path.startswith('/') and not path.startswith('//')
        headers = {'Origin': self.origin, 'X-CSRF-Token': self.csrf, 'Idempotency-Key': key or str(uuid.uuid4())}
        if revision is not None:
            headers['If-Match'] = '"' + str(revision) + '"'
        payload = body if isinstance(body, bytes) else None if body is None else json.dumps(body).encode()
        if payload is not None:
            headers['Content-Type'] = content_type or 'application/json'
        request = urllib.request.Request(self.origin + path, data=payload, headers=headers, method=method)
        try:
            with self.browser.open(request, timeout=20) as response:
                return json.load(response)
        except urllib.error.HTTPError as error:
            raise RuntimeError(method + ' ' + path + ': ' + str(error.code) + ' ' + error.read(4096).decode()) from None

    def download(self, url):
        parsed = urllib.parse.urlsplit(url)
        assert parsed.scheme == 'https' and parsed.netloc == 'evidence.teststudio.test'
        try:
            with self.browser.open(url, timeout=15) as response:
                return response.read()
        except urllib.error.URLError:
            raise RuntimeError('The private evidence download failed') from None

    def logout(self):
        assert self.call('POST', '/auth/logout', {})['signedOut'], 'The owned HTTP session was not revoked'
