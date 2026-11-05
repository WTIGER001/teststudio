package azuresandbox

func networkRule(name, direction, access, protocol, source, destination, port string, priority int) map[string]any {
	return map[string]any{"name": name, "properties": map[string]any{
		"direction": direction, "access": access, "protocol": protocol, "priority": priority,
		"sourceAddressPrefix": source, "destinationAddressPrefix": destination,
		"sourcePortRange": "*", "destinationPortRange": port,
	}}
}

func (p PlacementSettings) networkRules() []any {
	rules := []any{
		networkRule("sandbox-deny-in", "Inbound", "Deny", "*", "*", "*", "*", 4096),
		networkRule("sandbox-deny-metadata", "Outbound", "Deny", "*", "*", "AzurePlatformIMDS", "*", 100),
		networkRule("sandbox-dns-udp", "Outbound", "Allow", "Udp", p.SubnetPrefix, "AzurePlatformDNS", "53", 120),
		networkRule("sandbox-dns-tcp", "Outbound", "Allow", "Tcp", p.SubnetPrefix, "AzurePlatformDNS", "53", 121),
		networkRule("sandbox-private-https", "Outbound", "Allow", "Tcp", p.SubnetPrefix, "", "443", 300),
		networkRule("sandbox-deny-out", "Outbound", "Deny", "*", "*", "*", "*", 4096),
	}
	private := rules[4].(map[string]any)["properties"].(map[string]any)
	delete(private, "destinationAddressPrefix")
	private["destinationAddressPrefixes"] = append([]string(nil), p.Network.PrivateEndpointIPs...)
	return rules
}

func (p PlacementSettings) networkDefinitions(c Config) map[string]any {
	return map[string]any{
		p.SecurityGroup: map[string]any{"location": c.Region, "properties": map[string]any{"flushConnection": true, "securityRules": p.networkRules()}},
		p.RouteTable: map[string]any{"location": c.Region, "properties": map[string]any{"disableBgpRoutePropagation": true, "routes": []any{
			map[string]any{"name": "sandbox-egress", "properties": map[string]any{"addressPrefix": "0.0.0.0/0", "nextHopType": "VirtualAppliance", "nextHopIpAddress": p.Network.FirewallIP}},
		}}},
	}
}

func (s Settings) NetworkDefinitions() (map[string]any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s.Placement.networkDefinitions(s.Worker), nil
}
