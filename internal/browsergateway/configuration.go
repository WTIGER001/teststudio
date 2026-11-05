package browsergateway

import (
	"github.com/wtiger001/teststudio/internal/apicredential"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
)

type Configuration struct {
	Target     browserpolicy.Target `json:"target"`
	Credential *apicredential.Value `json:"credential,omitempty"`
	Remote     *RemoteSession       `json:"remote,omitempty"`
}

func (c Configuration) String() string   { return "private gateway configuration" }
func (c Configuration) GoString() string { return c.String() }
