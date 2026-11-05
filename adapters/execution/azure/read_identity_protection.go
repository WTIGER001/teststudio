package azureworker

import (
	"context"
	"time"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func verifyReadIdentityProtection(s ReadIdentitySettings, actual a.DenyAssignment) error {
	return verifyResourceDeny(s.identityProtection(), actual, "*/read", "Microsoft.Authorization/locks/delete")
}

func (c *SDKReadIdentityReader) Protection(ctx context.Context, id string) (a.DenyAssignment, error) {
	if id != c.settings.ProtectionAssignment {
		return a.DenyAssignment{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	response, err := c.denies.GetByID(ctx, id, nil)
	return response.DenyAssignment, sdkError(err, false)
}
