package ecsworker

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func gatewayPlanFixture(t *testing.T) (Config, GatewayConfig, hostedexecution.Job, workerexchange.GatewayTicket, *Plan) {
	t.Helper()
	config, job, _ := fixture(t, "go", "")
	job.Reference = (&hostedexecution.Executor{Name: "ecs", Runner: "go", Environment: true}).Reference(job.Reference)
	gateway := GatewayConfig{Image: "registry.example.test/studio/gateway@sha256:" + strings.Repeat("b", 64), Origin: "https://gateway.studio.example.test:8044", ExecutionRole: "arn:aws:iam::123456789012:role/studio-gateway-pull", Subnets: []string{"subnet-cccccccc"}, SecurityGroup: "sg-bbbbbbbb", CertificateSecret: "arn:aws:secretsmanager:us-east-1:123456789012:secret:studio/gateway/certificate-AbCdEf", KeySecret: "arn:aws:secretsmanager:us-east-1:123456789012:secret:studio/gateway/key-GhIjKl"}
	initialization := workerexchange.GatewayTicket{Execution: job.Reference.ID, Token: base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("g", 32))), Deadline: job.Ticket.Deadline}
	plan, err := NewGatewayPlan(config, gateway, job, initialization, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return config, gateway, job, initialization, plan
}

func TestECSGatewayPlanSeparatesNetworkIdentitySecretsAndIndependentDeadline(t *testing.T) {
	config, gateway, job, initialization, plan := gatewayPlanFixture(t)
	actual := definition(t, plan)
	if err := plan.VerifyDefinition(actual); err != nil {
		t.Fatal(err)
	}
	value := plan.Definition()
	container := value.ContainerDefinitions[0]
	if aws.ToString(value.ExecutionRoleArn) != gateway.ExecutionRole || value.TaskRoleArn != nil || len(value.Volumes) != 0 || len(container.MountPoints) != 0 || len(container.Environment) != 0 || len(container.Secrets) != 2 || aws.ToString(container.Name) != "gateway" || container.HealthCheck.Command[2] != "--ready-remote" || aws.ToString(value.Cpu) != "512" || aws.ToString(value.Memory) != "1024" {
		t.Fatal("gateway inherited worker source, data, credentials, or execution resources")
	}
	launch, err := plan.Launch(aws.ToString(actual.TaskDefinition.TaskDefinitionArn))
	if err != nil {
		t.Fatal(err)
	}
	network := launch.NetworkConfiguration.AwsvpcConfiguration
	if network.AssignPublicIp != types.AssignPublicIpDisabled || len(network.Subnets) != 1 || network.Subnets[0] != gateway.Subnets[0] || len(network.SecurityGroups) != 1 || network.SecurityGroups[0] != gateway.SecurityGroup || launch.EnableExecuteCommand {
		t.Fatal("gateway did not retain its separate private network identity")
	}
	fields := launch.Overrides.ContainerOverrides[0].Environment
	if len(fields) != 2 || aws.ToString(fields[0].Name) != "TESTSTUDIO_GATEWAY_EXECUTION" || aws.ToString(fields[1].Name) != "TESTSTUDIO_GATEWAY_TOKEN" || aws.ToString(fields[1].Value) != initialization.Token {
		t.Fatal("gateway bootstrap did not use its separate initialization authority")
	}
	deadline, err := plan.Deadline(taskARN(plan))
	if err != nil || aws.ToString(deadline.Name) != job.Reference.Gateway || aws.ToString(deadline.Target.RoleArn) != config.DeadlineRole || strings.Contains(aws.ToString(deadline.Target.Input), initialization.Token) {
		t.Fatal("gateway lacks its independent scoped deadline", err)
	}
	data, _ := json.Marshal(deadline)
	var observed scheduler.GetScheduleOutput
	json.Unmarshal(data, &observed)
	observed.Arn = aws.String("arn:aws:scheduler:" + config.Region + ":" + config.Account + ":schedule/" + config.ScheduleGroup + "/" + job.Reference.Gateway)
	if err := plan.VerifyDeadline(taskARN(plan), &observed); err != nil {
		t.Fatal(err)
	}
	proof, _ := json.Marshal(plan.Proof())
	definitionBytes, _ := json.Marshal(value)
	if strings.Contains(string(proof)+string(definitionBytes), initialization.Token) || strings.Contains(string(proof)+string(definitionBytes), job.Ticket.Token) {
		t.Fatal("raw execution authority entered durable gateway proof")
	}
	restored, err := Restore(config, plan.Proof())
	if err != nil || restored.VerifyDefinition(actual) != nil || restored.VerifyDeadline(taskARN(plan), &observed) != nil {
		t.Fatal("gateway recovery could not prove its original resources", err)
	}
	if _, err := restored.Launch(aws.ToString(actual.TaskDefinition.TaskDefinitionArn)); err == nil {
		t.Fatal("a durable proof could launch another gateway")
	}
	changed := plan.Proof()
	changed.Gateway.KeySecret = changed.Gateway.CertificateSecret
	if _, err := Restore(config, changed); err == nil {
		t.Fatal("a changed certificate reference retained gateway ownership")
	}
	if id, err := plan.GatewayInstance(taskARN(plan)); err != nil || id != strings.Repeat("a", 32) {
		t.Fatal("gateway task metadata and provider identity cannot agree", err)
	}
}

func TestECSGatewayTaskOwnershipAndReadinessRejectDrift(t *testing.T) {
	_, _, _, _, plan := gatewayPlanFixture(t)
	defined := definition(t, plan)
	launch, _ := plan.Launch(aws.ToString(defined.TaskDefinition.TaskDefinitionArn))
	value := types.Task{TaskArn: aws.String(taskARN(plan)), ClusterArn: aws.String(plan.config.Cluster), TaskDefinitionArn: defined.TaskDefinition.TaskDefinitionArn, StartedBy: launch.StartedBy, Group: launch.Group, Cpu: aws.String("512"), Memory: aws.String("1024"), PlatformVersion: aws.String("1.4.0"), PlatformFamily: aws.String("Linux"), LaunchType: types.LaunchTypeFargate, LastStatus: aws.String("RUNNING"), Overrides: launch.Overrides, Tags: launch.Tags, Containers: []types.Container{{Name: aws.String("gateway"), Image: aws.String(plan.job.Image)}}}
	if err := plan.verifyTask(value, aws.ToString(defined.TaskDefinition.TaskDefinitionArn)); err != nil {
		t.Fatal(err)
	}
	value.Containers[0].NetworkBindings = []types.NetworkBinding{{BindIP: aws.String("0.0.0.0"), ContainerPort: aws.Int32(8044), HostPort: aws.Int32(8044), Protocol: types.TransportProtocolTcp}}
	if err := plan.verifyTask(value, aws.ToString(defined.TaskDefinition.TaskDefinitionArn)); err != nil {
		t.Fatal("the gateway's fixed transport binding was rejected", err)
	}
	status, err := plan.taskStatus(value, false)
	if err != nil || !status.AdmissionPending {
		t.Fatal("unknown gateway health released activation", err)
	}
	value.HealthStatus = types.HealthStatusHealthy
	if status, err = plan.taskStatus(value, false); err != nil || status.AdmissionPending || status.InfrastructureFailure {
		t.Fatal("healthy gateway did not retain its native status", err)
	}
	value.HealthStatus = types.HealthStatusUnhealthy
	if status, err = plan.taskStatus(value, false); err != nil || !status.InfrastructureFailure {
		t.Fatal("unhealthy gateway did not fail its infrastructure boundary", err)
	}
	for _, mutate := range []func(*types.Task){
		func(v *types.Task) {
			v.Overrides.ContainerOverrides[0].Environment[1].Value = aws.String("another-initialization-ticket")
		},
		func(v *types.Task) { v.Overrides.ContainerOverrides[0].Name = aws.String("runner") },
		func(v *types.Task) {
			v.Containers[0].NetworkBindings = []types.NetworkBinding{{ContainerPort: aws.Int32(8080), HostPort: aws.Int32(8080), Protocol: types.TransportProtocolTcp}}
		},
		func(v *types.Task) { v.Cpu = aws.String("2048") },
	} {
		copy := copyValue(value)
		mutate(&copy)
		if plan.verifyTask(copy, aws.ToString(defined.TaskDefinition.TaskDefinitionArn)) == nil {
			t.Fatal("changed gateway task passed immutable ownership checks")
		}
	}
}

func TestECSGatewayPlanRejectsSharedRolesPoolsAndForeignCertificateReferences(t *testing.T) {
	config, gateway, job, initialization, _ := gatewayPlanFixture(t)
	for _, mutate := range []func(*GatewayConfig){
		func(g *GatewayConfig) { g.ExecutionRole = config.ExecutionRole },
		func(g *GatewayConfig) { g.SecurityGroup = config.SecurityGroups[0] },
		func(g *GatewayConfig) { g.Subnets = config.Subnets },
		func(g *GatewayConfig) {
			g.CertificateSecret = strings.Replace(g.CertificateSecret, config.Account, "999999999999", 1)
		},
		func(g *GatewayConfig) { g.KeySecret = g.CertificateSecret },
		func(g *GatewayConfig) { g.Origin = "https://10.1.2.3:8044" },
	} {
		changed := gateway
		mutate(&changed)
		if _, err := NewGatewayPlan(config, changed, job, initialization, time.Now()); err == nil {
			t.Fatal("an unapproved gateway boundary was accepted")
		}
	}
	initialization.Token = job.Ticket.Token
	if _, err := NewGatewayPlan(config, gateway, job, initialization, time.Now()); err == nil {
		t.Fatal("worker authority became gateway initialization authority")
	}
}
