package azureworker

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
)

func (s PreparationSettings) PreparationFileZoneDefinition() (p.PrivateZone, error) {
	if err := s.ValidateNative(); err != nil {
		return p.PrivateZone{}, err
	}
	return p.PrivateZone{Location: to.Ptr("global")}, nil
}

func (s PreparationSettings) PreparationFileLinkDefinition() (p.VirtualNetworkLink, error) {
	if err := s.ValidateNative(); err != nil {
		return p.VirtualNetworkLink{}, err
	}
	return s.Network.binding(s).linkDefinition(), nil
}

func (s PreparationSettings) PreparationFileRecordDefinitions() (map[string]p.RecordSet, error) {
	if err := s.ValidateNative(); err != nil {
		return nil, err
	}
	return s.Network.binding(s).recordDefinitions(), nil
}
