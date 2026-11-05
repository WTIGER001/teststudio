package ecsworker

import (
	"encoding/base64"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/ecsidentity"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

type GatewayConfig struct {
	Image             string   `json:"image"`
	Origin            string   `json:"origin"`
	ExecutionRole     string   `json:"executionRole"`
	Subnets           []string `json:"subnets"`
	SecurityGroup     string   `json:"securityGroup"`
	CertificateSecret string   `json:"certificateSecret"`
	KeySecret         string   `json:"keySecret"`
}

var secretName = regexp.MustCompile(`^[A-Za-z0-9/_+=.@-]{1,512}-[A-Za-z0-9]{6}$`)

func validateGateway(config Config, gateway GatewayConfig) error {
	origin, err := url.Parse(gateway.Origin)
	if Validate(config) != nil || err != nil || browsergateway.ValidateRemoteEndpoint(gateway.Origin, "") != nil || net.ParseIP(origin.Hostname()) != nil || len(origin.Hostname()) > 253 || !exchangeHostname.MatchString(origin.Hostname()) || origin.Port() != "8044" || !immutable.MatchString(gateway.Image) || len(gateway.Image) > 1024 || strings.Contains(gateway.Image, "..") || !config.validRole(gateway.ExecutionRole) || gateway.ExecutionRole == config.ExecutionRole || gateway.ExecutionRole == config.DeadlineRole || !securityGroup.MatchString(gateway.SecurityGroup) || slices.Contains(config.SecurityGroups, gateway.SecurityGroup) || len(gateway.Subnets) < 1 || len(gateway.Subnets) > 16 {
		return hostedexecution.ErrConfiguration
	}
	seen := map[string]bool{}
	for _, id := range gateway.Subnets {
		if !subnet.MatchString(id) || seen[id] || slices.Contains(config.Subnets, id) {
			return hostedexecution.ErrConfiguration
		}
		seen[id] = true
	}
	secretPrefix := "arn:aws:secretsmanager:" + config.Region + ":" + config.Account + ":secret:"
	for _, reference := range []string{gateway.CertificateSecret, gateway.KeySecret} {
		if !strings.HasPrefix(reference, secretPrefix) || !secretName.MatchString(strings.TrimPrefix(reference, secretPrefix)) {
			return hostedexecution.ErrConfiguration
		}
	}
	if gateway.CertificateSecret == gateway.KeySecret {
		return hostedexecution.ErrConfiguration
	}
	return nil
}

func canonicalGateway(config GatewayConfig) GatewayConfig {
	config.Subnets = slices.Clone(config.Subnets)
	slices.Sort(config.Subnets)
	return config
}

func validGatewayReference(ref execution.Reference, runner, operation string) bool {
	if operation != "" || runner != "go" && runner != "playwright" {
		return false
	}
	expected := (&hostedexecution.Executor{Name: "ecs", Runner: runner, Environment: true}).Reference(ref)
	copy := ref
	copy.Gateway = ""
	copy.Profile = runner + "-ecs-isolated-v1"
	return validReference(copy, runner, "") && ref == expected
}

func NewGatewayPlan(config Config, gateway GatewayConfig, job hostedexecution.Job, initialization workerexchange.GatewayTicket, now time.Time) (*Plan, error) {
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(initialization.Token)
	if err != nil || len(decoded) != 32 || !ticket.MatchString(job.Ticket.Token) || job.Ticket.Execution != job.Reference.ID || initialization.Token == job.Ticket.Token || initialization.Execution != job.Reference.ID || !initialization.Deadline.Equal(job.Ticket.Deadline) || !initialization.Deadline.After(now) || initialization.Deadline.After(now.Add(2*time.Hour)) {
		return nil, hostedexecution.ErrConfiguration
	}
	job.Image = gateway.Image
	job.Ticket = workerexchange.Ticket{Execution: initialization.Execution, Token: initialization.Token, Deadline: initialization.Deadline}
	return newTaskPlan(config, job, fingerprint(initialization.Token), &gateway)
}

func (p *Plan) gatewayDefinition(tagged bool) *ecs.RegisterTaskDefinitionInput {
	architecture := types.CPUArchitectureX8664
	if p.config.Architecture == "arm64" {
		architecture = types.CPUArchitectureArm64
	}
	value := &ecs.RegisterTaskDefinitionInput{
		Family: aws.String(p.family), Cpu: aws.String("512"), Memory: aws.String("1024"),
		ExecutionRoleArn: aws.String(p.gateway.ExecutionRole), EnableFaultInjection: aws.Bool(false),
		NetworkMode: types.NetworkModeAwsvpc, RequiresCompatibilities: []types.Compatibility{types.CompatibilityFargate},
		RuntimePlatform: &types.RuntimePlatform{CpuArchitecture: architecture, OperatingSystemFamily: types.OSFamilyLinux},
		ContainerDefinitions: []types.ContainerDefinition{{
			Name: aws.String("gateway"), Image: aws.String(p.gateway.Image), Essential: aws.Bool(true), User: aws.String("1000:1000"),
			EntryPoint: []string{"/gateway"}, Command: []string{"--remote-ecs-exchange=" + p.config.ExchangeOrigin}, WorkingDirectory: aws.String("/"),
			ReadonlyRootFilesystem: aws.Bool(true), Privileged: aws.Bool(false), Interactive: aws.Bool(false), PseudoTerminal: aws.Bool(false),
			StopTimeout: aws.Int32(10), RestartPolicy: &types.ContainerRestartPolicy{Enabled: aws.Bool(false)},
			LinuxParameters: &types.LinuxParameters{InitProcessEnabled: aws.Bool(true), Capabilities: &types.KernelCapabilities{Drop: []string{"ALL"}}},
			PortMappings:    []types.PortMapping{{ContainerPort: aws.Int32(8044), HostPort: aws.Int32(8044), Protocol: types.TransportProtocolTcp}},
			HealthCheck:     &types.HealthCheck{Command: []string{"CMD", "/gateway", "--ready-remote"}, Interval: aws.Int32(5), Timeout: aws.Int32(2), Retries: aws.Int32(2), StartPeriod: aws.Int32(10)},
			Secrets:         []types.Secret{{Name: aws.String("TESTSTUDIO_GATEWAY_TLS_CERTIFICATE"), ValueFrom: aws.String(p.gateway.CertificateSecret)}, {Name: aws.String("TESTSTUDIO_GATEWAY_TLS_KEY"), ValueFrom: aws.String(p.gateway.KeySecret)}},
		}},
	}
	if tagged {
		value.Tags = p.tags()
	}
	return value
}

func (p *Plan) containerName() string {
	if p.gateway != nil {
		return "gateway"
	}
	return "runner"
}

func (p *Plan) taskResources() (string, string) {
	if p.gateway != nil {
		return "512", "1024"
	}
	return "2048", "4096"
}

func (p *Plan) resourceName() string {
	if p.gateway != nil {
		return p.job.Reference.Gateway
	}
	return p.job.Reference.Container
}

func (p *Plan) validBindings(container types.Container) bool {
	if p.gateway == nil {
		return len(container.NetworkBindings) == 0
	}
	if len(container.NetworkBindings) > 1 {
		return false
	}
	for _, binding := range container.NetworkBindings {
		if aws.ToInt32(binding.ContainerPort) != 8044 || aws.ToInt32(binding.HostPort) != 8044 || binding.Protocol != types.TransportProtocolTcp || aws.ToString(binding.ContainerPortRange) != "" || aws.ToString(binding.HostPortRange) != "" {
			return false
		}
		if ip := aws.ToString(binding.BindIP); ip != "" && ip != "0.0.0.0" && (len(container.NetworkInterfaces) != 1 || ip != aws.ToString(container.NetworkInterfaces[0].PrivateIpv4Address)) {
			return false
		}
	}
	return true
}

func (p *Plan) GatewayInstance(taskARN string) (string, error) {
	if p.gateway == nil || !p.validTaskARN(taskARN) {
		return "", hostedexecution.ErrOwnership
	}
	value, err := ecsidentity.TaskInstance(taskARN)
	if err != nil {
		return "", hostedexecution.ErrOwnership
	}
	return value, nil
}
