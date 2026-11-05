package ecsidentity

import (
	"errors"
	"regexp"
)

const Protocol = "ecs-task/v1"

var task = regexp.MustCompile(`^arn:aws:ecs:[a-z]{2}(?:-[a-z0-9]+)+-[0-9]+:[0-9]{12}:task/[A-Za-z0-9_-]{1,255}/([a-f0-9]{32})$`)
var ErrIdentity = errors.New("ECS task identity is unavailable or invalid")

func TaskInstance(arn string) (string, error) {
	fields := task.FindStringSubmatch(arn)
	if len(fields) != 2 {
		return "", ErrIdentity
	}
	return fields[1], nil
}
