import argparse
import json
import os
from pathlib import Path
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser(description='Create the dedicated foundation administrator without exposing its password. Preserve existing accounts.')
    for name in ['profile', 'region', 'account', 'stack', 'output']:
        parser.add_argument('--' + name, required=True)
    args = parser.parse_args()
    base = ['aws', '--profile', args.profile, '--region', args.region, '--no-cli-pager']

    def call(arguments, value=None):
        private = None
        try:
            if value is not None:
                fd, private = tempfile.mkstemp(prefix='teststudio-identity-', suffix='.json')
                os.fchmod(fd, 0o600)
                with os.fdopen(fd, 'w') as request:
                    json.dump(value, request)
                arguments = arguments + ['--cli-input-json', 'file://' + private]
            result = subprocess.run(base + arguments, capture_output=True, text=True)
            if result.returncode:
                raise RuntimeError('AWS ' + ' '.join(arguments[:2]) + ' failed')
            return json.loads(result.stdout) if result.stdout.strip() else {}
        finally:
            if private:
                Path(private).unlink()

    assert call(['sts', 'get-caller-identity'])['Account'] == args.account, 'Unexpected AWS account'
    stack = call(['cloudformation', 'describe-stacks', '--stack-name', args.stack])['Stacks'][0]
    assert stack['StackStatus'] in ['CREATE_COMPLETE', 'UPDATE_COMPLETE'], 'The foundation is not ready'
    assert {'Key': 'Application', 'Value': 'TestStudio'} in stack.get('Tags', []), 'TestStudio ownership is required'
    outputs = {item['OutputKey']: item['OutputValue'] for item in stack['Outputs']}
    pool = outputs['UserPoolId']
    username = 'studioadmin'
    response = subprocess.run(base + ['cognito-idp', 'admin-get-user', '--user-pool-id', pool, '--username', username], capture_output=True, text=True)
    if response.returncode:
        assert 'UserNotFoundException' in response.stderr, 'The initial account could not be inspected'
        secret = json.loads(call(['secretsmanager', 'get-secret-value', '--secret-id', outputs['AdministratorSecretArn']])['SecretString'])
        assert secret['username'] == username
        call(['cognito-idp', 'admin-create-user'], {'UserPoolId': pool, 'Username': username, 'TemporaryPassword': secret['password'], 'MessageAction': 'SUPPRESS'})
        call(['cognito-idp', 'admin-set-user-password'], {'UserPoolId': pool, 'Username': username, 'Password': secret['password'], 'Permanent': True})
        secret = None
    user = call(['cognito-idp', 'admin-get-user', '--user-pool-id', pool, '--username', username])
    assert user['UserStatus'] == 'CONFIRMED', 'An existing unconfirmed account requires explicit operator recovery; its password remains unchanged'
    receipt = {'poolId': pool, 'username': username, 'subject': next(item['Value'] for item in user['UserAttributes'] if item['Name'] == 'sub'), 'status': user['UserStatus'], 'issuer': outputs['OidcIssuer'], 'credentialSecret': outputs['AdministratorSecretArn']}
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(receipt, indent=2) + '\n')
    print(json.dumps(receipt))


if __name__ == '__main__':
    main()
