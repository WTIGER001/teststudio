package ecsworker

import (
	"net/netip"
	"net/url"
	"regexp"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type NetworkSettings struct {
	VPC                 string `json:"vpc"`
	ControllerVPC       string `json:"controllerVpc"`
	ExchangeGroup       string `json:"exchangeGroup"`
	RegistryGroup       string `json:"registryGroup"`
	LoadBalancerGroup   string `json:"loadBalancerGroup"`
	ControllerGroup     string `json:"controllerGroup"`
	ExchangeEndpoint    string `json:"exchangeEndpoint"`
	ExchangeService     string `json:"exchangeService"`
	APIEndpoint         string `json:"apiEndpoint"`
	DKREndpoint         string `json:"dkrEndpoint"`
	S3Endpoint          string `json:"s3Endpoint"`
	S3PrefixList        string `json:"s3PrefixList"`
	RepositoryPrefix    string `json:"repositoryPrefix"`
	LoadBalancer        string `json:"loadBalancer"`
	TargetGroup         string `json:"targetGroup"`
	FirewallGroup       string `json:"firewallGroup"`
	FirewallAssociation string `json:"firewallAssociation"`
	AllowList           string `json:"allowList"`
	BlockList           string `json:"blockList"`
}

var vpcID = regexp.MustCompile(`^vpc-(?:[a-f0-9]{8}|[a-f0-9]{17})$`)
var endpointID = regexp.MustCompile(`^vpce-[a-f0-9]{17}$`)
var prefixListID = regexp.MustCompile(`^pl-[a-f0-9]{8,17}$`)
var repositoryPrefix = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*$`)
var endpointServiceID = regexp.MustCompile(`^vpce-svc-[a-f0-9]{17}$`)
var networkBalancer = regexp.MustCompile(`^loadbalancer/net/[A-Za-z0-9](?:[A-Za-z0-9-]{0,30}[A-Za-z0-9])?/[a-f0-9]{16}$`)
var networkTarget = regexp.MustCompile(`^targetgroup/[A-Za-z0-9](?:[A-Za-z0-9-]{0,30}[A-Za-z0-9])?/[a-f0-9]{16}$`)
var exchangeHostname = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$`)
var resolverID = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)

func (s NetworkSettings) Bind(c Config) (Config, error) {
	if Validate(c) != nil || !vpcID.MatchString(s.VPC) || !vpcID.MatchString(s.ControllerVPC) || s.VPC == s.ControllerVPC || !prefixListID.MatchString(s.S3PrefixList) || !repositoryPrefix.MatchString(s.RepositoryPrefix) || len(s.RepositoryPrefix) > 255 {
		return c, hostedexecution.ErrConfiguration
	}
	u, _ := url.Parse(c.ExchangeOrigin)
	_, addressError := netip.ParseAddr(u.Hostname())
	if addressError == nil || len(u.Hostname()) > 253 || !exchangeHostname.MatchString(u.Hostname()) || u.Port() != "" && u.Port() != "443" {
		return c, hostedexecution.ErrConfiguration
	}
	seen := map[string]bool{c.SecurityGroups[0]: true}
	for _, group := range []string{s.ExchangeGroup, s.RegistryGroup, s.LoadBalancerGroup, s.ControllerGroup} {
		if !securityGroup.MatchString(group) || seen[group] {
			return c, hostedexecution.ErrConfiguration
		}
		seen[group] = true
	}
	for _, id := range []string{s.ExchangeEndpoint, s.APIEndpoint, s.DKREndpoint, s.S3Endpoint} {
		if !endpointID.MatchString(id) || seen[id] {
			return c, hostedexecution.ErrConfiguration
		}
		seen[id] = true
	}
	for _, field := range []struct{ value, prefix string }{{s.FirewallGroup, "rslvr-frg-"}, {s.FirewallAssociation, "rslvr-frgassoc-"}, {s.AllowList, "rslvr-fdl-"}, {s.BlockList, "rslvr-fdl-"}} {
		if !resolverID.MatchString(field.value) || !strings.HasPrefix(field.value, field.prefix) || seen[field.value] {
			return c, hostedexecution.ErrConfiguration
		}
		seen[field.value] = true
	}
	base := "arn:aws:elasticloadbalancing:" + c.Region + ":" + c.Account + ":"
	if !strings.HasPrefix(s.LoadBalancer, base) || !networkBalancer.MatchString(strings.TrimPrefix(s.LoadBalancer, base)) || !strings.HasPrefix(s.TargetGroup, base) || !networkTarget.MatchString(strings.TrimPrefix(s.TargetGroup, base)) {
		return c, hostedexecution.ErrConfiguration
	}
	if !strings.HasPrefix(s.ExchangeService, "com.amazonaws.vpce."+c.Region+".") || !endpointServiceID.MatchString(strings.TrimPrefix(s.ExchangeService, "com.amazonaws.vpce."+c.Region+".")) {
		return c, hostedexecution.ErrConfiguration
	}
	identity := fingerprint([]any{"ecs-private-network/v1", s})
	if c.NetworkProfile != "" && c.NetworkProfile != identity {
		return c, hostedexecution.ErrConfiguration
	}
	c.NetworkProfile = identity
	return canonical(c), nil
}

func (s NetworkSettings) Domains(c Config) []string {
	u, _ := url.Parse(c.ExchangeOrigin)
	return []string{u.Hostname(), c.Account + ".dkr.ecr." + c.Region + ".amazonaws.com", "api.ecr." + c.Region + ".amazonaws.com", "prod-" + c.Region + "-starport-layer-bucket.s3." + c.Region + ".amazonaws.com", "prod-" + c.Region + "-starport-layer-bucket.s3.amazonaws.com"}
}

func (s NetworkSettings) ECRPolicy(c Config) map[string]any {
	arn := "arn:aws:ecr:" + c.Region + ":" + c.Account + ":repository/" + s.RepositoryPrefix
	condition := map[string]any{"ArnEquals": map[string]string{"aws:PrincipalArn": c.ExecutionRole}}
	return map[string]any{"Version": "2012-10-17", "Statement": []any{
		map[string]any{"Sid": "RuntimePull", "Effect": "Allow", "Principal": "*", "Action": []string{"ecr:BatchCheckLayerAvailability", "ecr:BatchGetImage", "ecr:GetDownloadUrlForLayer"}, "Resource": []string{arn, arn + "/*"}, "Condition": condition},
		map[string]any{"Sid": "RuntimeAuthorization", "Effect": "Allow", "Principal": "*", "Action": "ecr:GetAuthorizationToken", "Resource": "*", "Condition": condition},
	}}
}

func (s NetworkSettings) S3Policy(c Config) map[string]any {
	return map[string]any{"Version": "2012-10-17", "Statement": []any{map[string]any{"Sid": "RuntimeLayers", "Effect": "Allow", "Principal": "*", "Action": "s3:GetObject", "Resource": "arn:aws:s3:::prod-" + c.Region + "-starport-layer-bucket/*"}}}
}
