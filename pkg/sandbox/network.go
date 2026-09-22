package sandbox

import "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"

// toCreateConfig converts an egress policy into the shape Create sends.
func (n *NetworkConfig) toCreateConfig() *api.SandboxNetworkConfig {
	if n == nil {
		return nil
	}
	config := &api.SandboxNetworkConfig{AllowPublicTraffic: n.AllowPublicTraffic}
	if len(n.AllowOut) > 0 {
		allow := n.AllowOut
		config.AllowOut = &allow
	}
	if len(n.DenyOut) > 0 {
		deny := n.DenyOut
		config.DenyOut = &deny
	}
	if n.MaskRequestHost != "" {
		config.MaskRequestHost = &n.MaskRequestHost
	}
	return config
}

// NetworkUpdate replaces a running sandbox's egress policy.
//
// It is a smaller thing than NetworkConfig: the update endpoint accepts only
// the egress rules, not the public-traffic or host-masking settings that are
// fixed when the sandbox is created.
//
// The endpoint replaces rather than merges, so a field left nil is cleared.
type NetworkUpdate struct {
	// AllowOut permits egress to these CIDRs, IPs or domains.
	AllowOut []string

	// DenyOut blocks egress to these CIDRs or IPs. Domains are not accepted
	// here. Use AllTraffic to block everything.
	DenyOut []string

	// AllowInternetAccess permits egress. false behaves like denying
	// AllTraffic.
	AllowInternetAccess *bool
}

// toAPI converts an update into the generated type.
func (n NetworkUpdate) toAPI() api.SandboxNetworkUpdateConfig {
	config := api.SandboxNetworkUpdateConfig{AllowInternetAccess: n.AllowInternetAccess}
	if n.AllowOut != nil {
		allow := n.AllowOut
		config.AllowOut = &allow
	}
	if n.DenyOut != nil {
		deny := n.DenyOut
		config.DenyOut = &deny
	}
	return config
}

// networkFrom converts a reported egress policy back into this package's type.
func networkFrom(config *api.SandboxNetworkConfig) *NetworkConfig {
	if config == nil {
		return nil
	}
	network := &NetworkConfig{AllowPublicTraffic: config.AllowPublicTraffic}
	if config.AllowOut != nil {
		network.AllowOut = *config.AllowOut
	}
	if config.DenyOut != nil {
		network.DenyOut = *config.DenyOut
	}
	if config.MaskRequestHost != nil {
		network.MaskRequestHost = *config.MaskRequestHost
	}
	return network
}
