package azuresandbox

import (
	"regexp"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type DeadlineConfig struct {
	Worker        Config `json:"worker"`
	ResourceGroup string `json:"resourceGroup"`
	NamePrefix    string `json:"namePrefix"`
	StopIdentity  string `json:"stopIdentity"`
}

var deadlinePrefix = regexp.MustCompile(`^[a-z][a-z0-9-]{0,14}[a-z0-9]$`)

func (c DeadlineConfig) validate() error {
	if Validate(c.Worker) != nil || !groupName.MatchString(c.ResourceGroup) || strings.HasSuffix(c.ResourceGroup, ".") || strings.Contains(c.ResourceGroup, "..") || strings.EqualFold(c.ResourceGroup, c.Worker.ResourceGroup) || !deadlinePrefix.MatchString(c.NamePrefix) || !c.Worker.validPullIdentity(c.StopIdentity) {
		return hostedexecution.ErrConfiguration
	}
	return nil
}

func (c DeadlineConfig) profile() string {
	return hash([]any{"sandbox-deadline/v1", c, APIVersion, TokenScope})
}
func (c DeadlineConfig) name(s DeadlineSpec) string {
	return c.NamePrefix + "-stop-" + hash(s.Proof.Reference)[:24]
}

func (c DeadlineConfig) resourceID(s DeadlineSpec) string {
	return "/subscriptions/" + c.Worker.Subscription + "/resourceGroups/" + c.ResourceGroup + "/providers/Microsoft.Logic/workflows/" + c.name(s)
}

func (c DeadlineConfig) validateSpec(s DeadlineSpec) error {
	if c.validate() != nil || hash(s.Config) != hash(c.Worker) {
		return hostedexecution.ErrOwnership
	}
	_, err := Restore(c.Worker, s.Proof)
	return err
}
