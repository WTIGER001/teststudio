package azureworker

import (
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

type firewallFixture struct {
	t            *testing.T
	mu           sync.Mutex
	settings     FirewallSettings
	firewall     n.AzureFirewall
	policy       n.FirewallPolicy
	subnet       n.Subnet
	public       n.PublicIPAddress
	groups       []n.FirewallPolicyRuleCollectionGroup
	reads, pages int
	mode         string
}

func firewallTest(t *testing.T) (Config, FirewallSettings, *firewallFixture, *FirewallGuard) {
	t.Helper()
	c, perimeter, _, _, guard := perimeterTest(t)
	s := FirewallSettings{Perimeter: perimeter, Firewall: c.resourceID("Microsoft.Network", "azureFirewalls", "studio-firewall"), Policy: c.resourceID("Microsoft.Network", "firewallPolicies", "studio-worker-policy"), PublicIP: c.resourceID("Microsoft.Network", "publicIPAddresses", "studio-firewall-ip"), SubnetPrefix: "10.20.1.0/26"}
	f := &firewallFixture{t: t, settings: s}
	f.firewall, _ = s.FirewallDefinition(c)
	f.firewall.ID, f.firewall.Name, f.firewall.Type = to.Ptr(s.Firewall), to.Ptr("studio-firewall"), to.Ptr("Microsoft.Network/azureFirewalls")
	f.firewall.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	ip := f.firewall.Properties.IPConfigurations[0]
	ip.ID = to.Ptr(s.configuration())
	ip.Properties.PrivateIPAddress = to.Ptr(s.Perimeter.FirewallIP)
	ip.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	f.policy, _ = s.PolicyDefinition(c)
	f.policy.ID, f.policy.Name, f.policy.Type = to.Ptr(s.Policy), to.Ptr("studio-worker-policy"), to.Ptr("Microsoft.Network/firewallPolicies")
	f.policy.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	f.policy.Properties.Firewalls = []*n.SubResource{{ID: to.Ptr(s.Firewall)}}
	own, _ := s.RuleGroupDefinition(c)
	own.ID, own.Name, own.Type = to.Ptr(s.group()), to.Ptr("worker-platform"), to.Ptr("Microsoft.Network/firewallPolicies/ruleCollectionGroups")
	own.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	f.groups = []n.FirewallPolicyRuleCollectionGroup{own}
	for i, name := range []string{"DefaultDnatRuleCollectionGroup", "DefaultNetworkRuleCollectionGroup", "DefaultApplicationRuleCollectionGroup"} {
		f.groups = append(f.groups, n.FirewallPolicyRuleCollectionGroup{ID: to.Ptr(s.Policy + "/ruleCollectionGroups/" + name), Name: to.Ptr(name), Type: to.Ptr("Microsoft.Network/firewallPolicies/ruleCollectionGroups"), Properties: &n.FirewallPolicyRuleCollectionGroupProperties{Priority: to.Ptr(int32((i + 1) * 100)), ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded)}})
	}
	for _, group := range f.groups {
		f.policy.Properties.RuleCollectionGroups = append(f.policy.Properties.RuleCollectionGroups, &n.SubResource{ID: group.ID})
	}
	f.subnet = n.Subnet{ID: to.Ptr(s.subnet()), Name: to.Ptr("AzureFirewallSubnet"), Type: to.Ptr("Microsoft.Network/virtualNetworks/subnets"), Properties: &n.SubnetPropertiesFormat{AddressPrefix: to.Ptr(s.SubnetPrefix), DefaultOutboundAccess: to.Ptr(false), ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded)}}
	f.public = n.PublicIPAddress{ID: to.Ptr(s.PublicIP), Name: to.Ptr("studio-firewall-ip"), Type: to.Ptr("Microsoft.Network/publicIPAddresses"), Location: to.Ptr(c.Region), SKU: &n.PublicIPAddressSKU{Name: to.Ptr(n.PublicIPAddressSKUNameStandard), Tier: to.Ptr(n.PublicIPAddressSKUTierRegional)}, Properties: &n.PublicIPAddressPropertiesFormat{IPAddress: to.Ptr("203.0.113.10"), PublicIPAllocationMethod: to.Ptr(n.IPAllocationMethodStatic), PublicIPAddressVersion: to.Ptr(n.IPVersionIPv4), IPConfiguration: &n.IPConfiguration{ID: to.Ptr(s.configuration())}, ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded)}}
	reader, err := NewSDKFirewallReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	firewall, err := NewFirewallGuard(c, s, guard, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, firewall
}

func (f *firewallFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.Query().Get("api-version") != "2025-03-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("firewall SDK escaped the read boundary")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-provider-canary"}}), nil
	}
	if f.mode == "missing" {
		return deadlineResponse(r, 404, map[string]any{"error": map[string]any{"code": "NotFound"}}), nil
	}
	if f.mode == "redirect" {
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private")
		return response, nil
	}
	var object any
	switch r.URL.Path {
	case f.settings.Firewall:
		object = f.firewall
	case f.settings.Policy:
		object = f.policy
	case f.settings.subnet():
		object = f.subnet
	case f.settings.PublicIP:
		object = f.public
	case f.settings.Policy + "/ruleCollectionGroups":
		f.pages++
		if f.mode == "groups-missing" {
			return deadlineResponse(r, 404, map[string]any{"error": map[string]any{"code": "NotFound"}}), nil
		}
		page := map[string]any{"value": f.groups}
		switch f.mode {
		case "paged", "cycle":
			if r.URL.Query().Get("$skipToken") == "" {
				page["value"] = f.groups[:2]
				page["nextLink"] = "https://management.azure.com" + r.URL.Path + "?api-version=2025-03-01&%24skipToken=page-one"
			} else {
				page["value"] = f.groups[2:]
				if f.mode == "cycle" {
					page["nextLink"] = "https://management.azure.com" + r.URL.Path + "?api-version=2025-03-01&%24skipToken=page-one"
				}
			}
		case "foreign-page":
			page["nextLink"] = "https://foreign.invalid/private?api-version=2025-03-01&%24skipToken=one"
		case "foreign-parent":
			page["nextLink"] = "https://management.azure.com" + f.settings.Policy + "-foreign/ruleCollectionGroups?api-version=2025-03-01&%24skipToken=one"
		case "groups-hidden":
			page["value"] = []any{}
		case "endless":
			page["value"] = []any{}
			page["nextLink"] = "https://management.azure.com" + r.URL.Path + "?api-version=2025-03-01&%24skipToken=" + strconv.Itoa(f.pages)
		case "overflow":
			page["value"] = append(append([]n.FirewallPolicyRuleCollectionGroup{}, f.groups...), f.groups[0])
		}
		object = page
	default:
		f.t.Error("firewall reader requested an unconfigured resource")
		return deadlineResponse(r, 404, nil), nil
	}
	response := deadlineResponse(r, 200, object)
	if f.mode == "oversized" {
		response.ContentLength = 5 << 20
	}
	return response, nil
}

func (f *firewallFixture) allow() *n.ApplicationRule {
	return f.groups[0].Properties.RuleCollections[0].(*n.FirewallPolicyFilterRuleCollection).Rules[0].(*n.ApplicationRule)
}
