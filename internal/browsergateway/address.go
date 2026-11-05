package browsergateway

import (
	"net/netip"
	"strings"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
)

func ValidateRemoteEndpoint(origin, address string) error {
	canonical, err := browserpolicy.Origin(origin)
	if err != nil || canonical != origin || !strings.HasPrefix(origin, "https://") {
		return ErrRemoteConfiguration
	}
	if address != "" {
		ip, err := netip.ParseAddr(address)
		if err != nil || !ip.Is4() || ip.String() != address || !ip.IsPrivate() && !ip.IsLoopback() {
			return ErrRemoteConfiguration
		}
	}
	return nil
}
