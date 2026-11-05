package azuresandbox

func (p PrivateSettings) rules(s Settings) []any {
	allow := networkRule("sandbox-endpoint-https", "Inbound", "Allow", "Tcp", s.Placement.SubnetPrefix, "", "443", 100)
	props := allow["properties"].(map[string]any)
	delete(props, "destinationAddressPrefix")
	props["destinationAddressPrefixes"] = append([]string(nil), s.Placement.Network.PrivateEndpointIPs...)
	return []any{allow, networkRule("endpoint-deny-in", "Inbound", "Deny", "*", "*", "*", "*", 4096), networkRule("endpoint-deny-out", "Outbound", "Deny", "*", "*", "*", "*", 4096)}
}

func (s Settings) PrivateDNSDefinitions() (map[string]any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	result := map[string]any{}
	for _, zone := range s.Placement.Private.zoneBindings(s) {
		result[zone.ID] = map[string]any{"location": "global"}
		result[zone.ID+"/virtualNetworkLinks/"+zone.LinkName] = zone.LinkDefinition()
		for name, record := range zone.RecordDefinitions() {
			result[zone.ID+"/A/"+name] = record
		}
	}
	return result, nil
}
