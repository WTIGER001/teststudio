package ecsworker

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestFargatePlanningRejectsUnscopedPoolsAndUnboundedWork(t *testing.T) {
	mutations := []func(*Config){
		func(c *Config) { c.Cluster = strings.Replace(c.Cluster, c.Account, "999999999999", 1) },
		func(c *Config) { c.ExecutionRole = strings.Replace(c.ExecutionRole, c.Account, "999999999999", 1) },
		func(c *Config) { c.DeadlineRole = c.ExecutionRole },
		func(c *Config) { c.Platform = "LATEST" },
		func(c *Config) { c.ExchangeOrigin = "http://worker.example.test" },
		func(c *Config) { c.ExchangeOrigin += "/work" },
		func(c *Config) { c.Subnets = append(c.Subnets, c.Subnets[0]) },
		func(c *Config) { c.SecurityGroups = append(c.SecurityGroups, "sg-bbbbbbbb") },
		func(c *Config) { c.Architecture = "windows" },
	}
	for index, mutate := range mutations {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			c, job, _ := fixture(t, "go", "")
			mutate(&c)
			if _, err := NewPlan(c, job, time.Now()); err == nil {
				t.Fatal("unscoped or ambiguous worker pool was admitted")
			}
		})
	}
	c, job, p := fixture(t, "go", "")
	job.Ticket.Deadline = time.Now().Add(-time.Second)
	if _, err := NewPlan(c, job, time.Now()); err == nil {
		t.Fatal("expired work could launch")
	}
	job.Ticket.Deadline = time.Now().Add(3 * time.Hour)
	if _, err := NewPlan(c, job, time.Now()); err == nil {
		t.Fatal("work exceeded the maximum hosted wall time")
	}
	job = p.job
	job.Operation = "discover"
	if _, err := NewPlan(c, job, time.Now()); err == nil {
		t.Fatal("discovery could reuse an execution identity")
	}
}

func TestFargatePoolIdentityIgnoresSubnetOrderAndSeparatesRuntimeClasses(t *testing.T) {
	c, _, _ := fixture(t, "go", "")
	first, err := ExecutionProfile(c)
	if err != nil {
		t.Fatal(err)
	}
	c.Subnets[0], c.Subnets[1] = c.Subnets[1], c.Subnets[0]
	reordered, err := ExecutionProfile(c)
	if err != nil || first != reordered {
		t.Fatal("subnet ordering split the timing baseline", err)
	}
	c.Architecture = "amd64"
	changed, err := ExecutionProfile(c)
	if err != nil || changed == first {
		t.Fatal("different runtime architecture shared a timing pool", err)
	}
}
