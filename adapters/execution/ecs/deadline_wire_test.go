package ecsworker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

func TestFargateDeadlineTargetMatchesTheSignedStopTaskWireContract(t *testing.T) {
	config, _, plan := fixture(t, "go", "")
	deadline, err := plan.Deadline(taskARN(plan))
	if err != nil {
		t.Fatal(err)
	}
	var expected map[string]string
	if json.Unmarshal([]byte(*deadline.Target.Input), &expected) != nil || len(expected) != 3 {
		t.Fatal("deadline input is not the bounded stop-task request")
	}
	observed := make(chan map[string]string, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("X-Amz-Target") != "AmazonEC2ContainerServiceV20141113.StopTask" || r.Header.Get("Authorization") == "" {
			t.Error("stop-task request did not use the signed ECS API")
		}
		var fields map[string]string
		if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
			t.Error(err)
		}
		observed <- fields
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.Write([]byte(`{"task":{}}`))
	}))
	defer server.Close()
	client := ecs.NewFromConfig(aws.Config{Region: config.Region, HTTPClient: server.Client(), Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
		return aws.Credentials{AccessKeyID: "owned-local-access", SecretAccessKey: "owned-local-signing-canary"}, nil
	})}, func(options *ecs.Options) {
		options.BaseEndpoint = aws.String(server.URL)
		options.RetryMaxAttempts = 1
	})
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if _, err := client.StopTask(ctx, &ecs.StopTaskInput{Cluster: aws.String(config.Cluster), Task: aws.String(taskARN(plan)), Reason: aws.String("TestStudio worker deadline")}); err != nil {
		t.Fatal(err)
	}
	if actual := <-observed; !reflect.DeepEqual(actual, expected) {
		t.Fatal("universal target input differs from the ECS wire request", actual)
	}
}
