package ecsworker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type Config struct {
	Region         string   `json:"region"`
	Account        string   `json:"account"`
	Cluster        string   `json:"cluster"`
	ExecutionRole  string   `json:"executionRole"`
	DeadlineRole   string   `json:"deadlineRole"`
	ScheduleGroup  string   `json:"scheduleGroup"`
	FamilyPrefix   string   `json:"familyPrefix"`
	Architecture   string   `json:"architecture"`
	Platform       string   `json:"platformVersion"`
	Subnets        []string `json:"subnets"`
	SecurityGroups []string `json:"securityGroups"`
	ExchangeOrigin string   `json:"exchangeOrigin"`
}

var region = regexp.MustCompile(`^[a-z]{2}(?:-[a-z0-9]+)+-[0-9]+$`)
var account = regexp.MustCompile(`^[0-9]{12}$`)
var resourceName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,255}$`)
var prefix = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
var scheduleGroup = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)
var rolePath = regexp.MustCompile(`^[A-Za-z0-9_+=,.@/-]{1,512}$`)
var subnet = regexp.MustCompile(`^subnet-(?:[a-f0-9]{8}|[a-f0-9]{17})$`)
var securityGroup = regexp.MustCompile(`^sg-(?:[a-f0-9]{8}|[a-f0-9]{17})$`)
var immutable = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*@sha256:[a-f0-9]{64}$`)
var ticket = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)
var checksum = regexp.MustCompile(`^[a-f0-9]{64}$`)

func fingerprint(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (c Config) ecsPrefix() string { return "arn:aws:ecs:" + c.Region + ":" + c.Account + ":" }

func (c Config) clusterName() string { return strings.TrimPrefix(c.Cluster, c.ecsPrefix()+"cluster/") }

func (c Config) validRole(value string) bool {
	base := "arn:aws:iam::" + c.Account + ":role/"
	return strings.HasPrefix(value, base) && rolePath.MatchString(strings.TrimPrefix(value, base))
}

func Validate(c Config) error {
	u, err := url.Parse(c.ExchangeOrigin)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || !region.MatchString(c.Region) || !account.MatchString(c.Account) || !strings.HasPrefix(c.Cluster, c.ecsPrefix()+"cluster/") || !resourceName.MatchString(c.clusterName()) || !c.validRole(c.ExecutionRole) || !c.validRole(c.DeadlineRole) || c.ExecutionRole == c.DeadlineRole || !scheduleGroup.MatchString(c.ScheduleGroup) || !prefix.MatchString(c.FamilyPrefix) || c.Architecture != "arm64" && c.Architecture != "amd64" || c.Platform != "1.4.0" || len(c.Subnets) < 1 || len(c.Subnets) > 16 || len(c.SecurityGroups) != 1 {
		return hostedexecution.ErrConfiguration
	}
	seen := map[string]bool{}
	for _, id := range c.Subnets {
		if !subnet.MatchString(id) || seen[id] {
			return hostedexecution.ErrConfiguration
		}
		seen[id] = true
	}
	if !securityGroup.MatchString(c.SecurityGroups[0]) {
		return hostedexecution.ErrConfiguration
	}
	return nil
}

func canonical(c Config) Config {
	c.Subnets, c.SecurityGroups = slices.Clone(c.Subnets), slices.Clone(c.SecurityGroups)
	slices.Sort(c.Subnets)
	return c
}

func ExecutionProfile(c Config) (string, error) {
	if err := Validate(c); err != nil {
		return "", err
	}
	return "fargate-v1-" + fingerprint([]any{canonical(c), "cpu2048-memory4096-tmpfs1536"})[:20], nil
}
