package ecsworker

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func normalize(value any) any {
	switch value := value.(type) {
	case map[string]any:
		result := map[string]any{}
		for key, item := range value {
			if next := normalize(item); next != nil {
				result[key] = next
			}
		}
		if len(result) > 0 {
			return result
		}
	case []any:
		if len(value) > 0 {
			result := make([]any, len(value))
			for index, item := range value {
				result[index] = normalize(item)
			}
			return result
		}
	case nil:
	case bool:
		if value {
			return value
		}
	case string:
		if value != "" {
			return value
		}
	case float64:
		if value != 0 {
			return value
		}
	default:
		return value
	}
	return nil
}

func document(value any, omit ...string) any {
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var fields map[string]any
	if json.Unmarshal(data, &fields) != nil {
		return nil
	}
	for _, key := range omit {
		delete(fields, key)
	}
	return normalize(fields)
}

func (p *Plan) VerifyTags(actual []types.Tag) error {
	fields := map[string]string{}
	for _, tag := range actual {
		key := aws.ToString(tag.Key)
		if strings.HasPrefix(key, "aws:") {
			continue
		}
		if _, duplicate := fields[key]; duplicate || key == "" || tag.Value == nil {
			return hostedexecution.ErrOwnership
		}
		fields[key] = *tag.Value
	}
	if len(fields) != len(p.tags()) {
		return hostedexecution.ErrOwnership
	}
	for _, tag := range p.tags() {
		if fields[*tag.Key] != *tag.Value {
			return hostedexecution.ErrOwnership
		}
	}
	return nil
}

func (p *Plan) VerifyDefinition(actual *ecs.DescribeTaskDefinitionOutput) error {
	return p.verifyDefinition(actual, false)
}

func (p *Plan) verifyDefinition(actual *ecs.DescribeTaskDefinitionOutput, cleanup bool) error {
	if actual == nil || actual.TaskDefinition == nil || !p.validDefinitionARN(aws.ToString(actual.TaskDefinition.TaskDefinitionArn)) || actual.TaskDefinition.Revision < 1 || p.VerifyTags(actual.Tags) != nil {
		return hostedexecution.ErrOwnership
	}
	state := actual.TaskDefinition.Status
	if state != types.TaskDefinitionStatusActive && (!cleanup || state != types.TaskDefinitionStatusInactive && state != types.TaskDefinitionStatusDeleteInProgress) {
		return hostedexecution.ErrOwnership
	}
	if !strings.HasSuffix(*actual.TaskDefinition.TaskDefinitionArn, ":"+strconv.FormatInt(int64(actual.TaskDefinition.Revision), 10)) {
		return hostedexecution.ErrOwnership
	}
	expected := document(p.Definition(), "Tags")
	observed := document(actual.TaskDefinition, "TaskDefinitionArn", "Revision", "Status", "RegisteredAt", "RegisteredBy", "DeregisteredAt", "DeleteRequestedAt", "Compatibilities", "RequiresAttributes")
	if expected == nil || !reflect.DeepEqual(expected, observed) {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (p *Plan) VerifyDeadline(taskARN string, actual *scheduler.GetScheduleOutput) error {
	expected, err := p.Deadline(taskARN)
	if err != nil || actual == nil {
		return hostedexecution.ErrOwnership
	}
	arn := "arn:aws:scheduler:" + p.config.Region + ":" + p.config.Account + ":schedule/" + p.config.ScheduleGroup + "/" + p.job.Reference.Container
	if aws.ToString(actual.Arn) != arn {
		return hostedexecution.ErrOwnership
	}
	left := document(expected, "ClientToken")
	right := document(actual, "Arn", "CreationDate", "LastModificationDate", "ResultMetadata")
	if left == nil || !reflect.DeepEqual(left, right) {
		return hostedexecution.ErrOwnership
	}
	return nil
}
