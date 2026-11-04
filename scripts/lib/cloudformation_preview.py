import hashlib
import re


ABSENT = object()


class Preview:
    def __init__(self, template, parameters, references, attributes, region='us-east-1'):
        self.template = template
        self.parameters = {name: value['Default'] for name, value in template.get('Parameters', {}).items() if 'Default' in value}
        self.parameters.update(parameters)
        self.references = {**references, 'AWS::Region': region, 'AWS::AccountId': '123456789012', 'AWS::Partition': 'aws', 'AWS::URLSuffix': 'amazonaws.com', 'AWS::StackName': 'studio-workers-stack', 'AWS::NoValue': ABSENT}
        self.attributes = attributes
        for name, definition in template.get('Parameters', {}).items():
            if name not in self.parameters:
                raise ValueError('Missing preview parameter: ' + name)
            value = self.parameters[name]
            if 'AllowedValues' in definition and value not in definition['AllowedValues']:
                raise ValueError('Invalid preview parameter: ' + name)
            if 'AllowedPattern' in definition and re.fullmatch(definition['AllowedPattern'], str(value)) is None:
                raise ValueError('Invalid preview pattern: ' + name)
            for key, operation in [('MinLength', lambda a, b: len(a) >= b), ('MaxLength', lambda a, b: len(a) <= b), ('MinValue', lambda a, b: a >= b), ('MaxValue', lambda a, b: a <= b)]:
                if key in definition and not operation(value, definition[key]):
                    raise ValueError('Invalid preview bound: ' + name)
        for rule in template.get('Rules', {}).values():
            if 'RuleCondition' not in rule or self.resolve(rule['RuleCondition']):
                for assertion in rule['Assertions']:
                    if not self.resolve(assertion['Assert']):
                        raise ValueError(assertion['AssertDescription'])

    def condition(self, name):
        return self.resolve(self.template['Conditions'][name])

    def resolve(self, value):
        if isinstance(value, list):
            values = [self.resolve(item) for item in value]
            return [item for item in values if item is not ABSENT]
        if not isinstance(value, dict):
            return value
        if set(value) == {'Ref'}:
            name = value['Ref']
            if name in self.parameters:
                return self.parameters[name]
            return self.references[name]
        if set(value) == {'Fn::GetAtt'}:
            parts = value['Fn::GetAtt']
            return self.attributes['.'.join(parts) if isinstance(parts, list) else parts]
        if set(value) == {'Fn::If'}:
            condition, positive, negative = value['Fn::If']
            return self.resolve(positive if self.condition(condition) else negative)
        if set(value) == {'Fn::Equals'}:
            left, right = value['Fn::Equals']
            return self.resolve(left) == self.resolve(right)
        if set(value) == {'Fn::Not'}:
            return not self.resolve(value['Fn::Not'][0])
        if set(value) == {'Fn::And'}:
            return all(self.resolve(item) for item in value['Fn::And'])
        if set(value) == {'Fn::Or'}:
            return any(self.resolve(item) for item in value['Fn::Or'])
        if set(value) == {'Fn::Join'}:
            separator, items = value['Fn::Join']
            return separator.join(self.resolve(items))
        if set(value) == {'Fn::Sub'}:
            substitution = value['Fn::Sub']
            text, variables = substitution if isinstance(substitution, list) else (substitution, {})
            def substitute(match):
                name = match[1]
                if name.startswith('!'):
                    return '${' + name[1:] + '}'
                if name in variables:
                    return str(self.resolve(variables[name]))
                return str(self.resolve({'Fn::GetAtt': name}) if '.' in name else self.resolve({'Ref': name}))
            return re.sub(r'\$\{([^}]+)\}', substitute, text)
        result = {key: self.resolve(item) for key, item in value.items()}
        return {key: item for key, item in result.items() if item is not ABSENT}

    def properties(self, name):
        resource = self.template['Resources'][name]
        if 'Condition' in resource and not self.condition(resource['Condition']):
            return None
        return self.resolve(resource.get('Properties', {}))


def identifiers(template, region='us-east-1'):
    references, attributes = {}, {}
    account = '123456789012'
    fixture = {'WorkerVpc': 'vpc-aaaaaaaa', 'WorkerSubnetOne': 'subnet-aaaaaaaa', 'WorkerSubnetTwo': 'subnet-bbbbbbbb', 'WorkerGroup': 'sg-aaaaaaaa', 'ExchangeGroup': 'sg-bbbbbbbb', 'RegistryGroup': 'sg-cccccccc', 'ExchangeLoadBalancerGroup': 'sg-dddddddd', 'ExchangeEndpoint': 'vpce-11111111111111111', 'EcrApi': 'vpce-22222222222222222', 'EcrDocker': 'vpce-33333333333333333', 'ImageLayers': 'vpce-44444444444444444', 'ExchangeService': 'vpce-svc-55555555555555555', 'WorkerCluster': 'studio-workers', 'WorkerPullRole': 'studio-image-pull', 'DeadlineRole': 'studio-deadlines', 'Deadlines': 'studio-worker-deadlines', 'DnsRules': 'rslvr-frg-11111111111111111', 'DnsAssociation': 'rslvr-frgassoc-22222222222222222', 'AllowedDomains': 'rslvr-fdl-33333333333333333', 'BlockedDomains': 'rslvr-fdl-44444444444444444', 'ExchangeLoadBalancer': f'arn:aws:elasticloadbalancing:{region}:{account}:loadbalancer/net/worker/aaaaaaaaaaaaaaaa', 'ExchangeTarget': f'arn:aws:elasticloadbalancing:{region}:{account}:targetgroup/worker/bbbbbbbbbbbbbbbb', 'TaskSecurityGroup': 'sg-eeeeeeee', 'WorkerRoutes': 'rtb-aaaaaaaa'}
    for name, resource in template['Resources'].items():
        suffix = hashlib.sha256(name.encode()).hexdigest()[:17]
        kind = resource['Type']
        value = fixture.get(name, name.lower())
        if name not in fixture:
            if kind == 'AWS::IAM::Role':
                value = 'studio-' + name.lower()
            elif kind == 'AWS::SecretsManager::Secret':
                value = f'arn:aws:secretsmanager:{region}:{account}:secret:studio-{name.lower()}-fixture'
            elif kind == 'AWS::EFS::FileSystem':
                value = 'fs-' + suffix
            elif kind == 'AWS::EFS::AccessPoint':
                value = 'fsap-' + suffix
            elif kind == 'AWS::EC2::SecurityGroup':
                value = 'sg-' + suffix
        references[name] = value
        attributes[name + '.Id'] = value
        if kind == 'AWS::IAM::Role':
            attributes[name + '.Arn'] = f'arn:aws:iam::{account}:role/{value}'
        elif kind == 'AWS::ECS::Cluster':
            attributes[name + '.Arn'] = f'arn:aws:ecs:{region}:{account}:cluster/{value}'
        elif kind == 'AWS::Scheduler::ScheduleGroup':
            attributes[name + '.Arn'] = f'arn:aws:scheduler:{region}:{account}:schedule-group/{value}'
        elif kind.startswith('AWS::Route53Resolver::'):
            resource_name = {'AWS::Route53Resolver::FirewallDomainList': 'firewall-domain-list', 'AWS::Route53Resolver::FirewallRuleGroup': 'firewall-rule-group', 'AWS::Route53Resolver::FirewallRuleGroupAssociation': 'firewall-rule-group-association'}[kind]
            attributes[name + '.Arn'] = f'arn:aws:route53resolver:{region}:{account}:{resource_name}/{value}'
        elif kind == 'AWS::Logs::LogGroup':
            attributes[name + '.Arn'] = f'arn:aws:logs:{region}:{account}:log-group:{value}:*'
        elif kind == 'AWS::EFS::FileSystem':
            attributes[name + '.Arn'] = f'arn:aws:elasticfilesystem:{region}:{account}:file-system/{value}'
        elif kind == 'AWS::EFS::AccessPoint':
            attributes[name + '.Arn'] = f'arn:aws:elasticfilesystem:{region}:{account}:access-point/{value}'
        elif kind == 'AWS::ECR::Repository':
            repository = 'teststudio/' + ('go' if name == 'GoRepository' else 'playwright')
            references[name] = repository
            attributes[name + '.Arn'] = f'arn:aws:ecr:{region}:{account}:repository/{repository}'
            attributes[name + '.RepositoryUri'] = f'{account}.dkr.ecr.{region}.amazonaws.com/{repository}'
    return references, attributes
