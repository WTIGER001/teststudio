package kubernetes

import (
	"context"
	"net/http"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func selector(peer Peer) map[string]any {
	return map[string]any{"namespaceSelector": map[string]any{"matchLabels": map[string]string{"kubernetes.io/metadata.name": peer.Namespace}}, "podSelector": map[string]any{"matchLabels": peer.Labels}}
}

func (p *Provider) policy() map[string]any {
	port := p.config.ExchangePort
	if port == 0 {
		port = 443
	}
	return map[string]any{
		"podSelector": map[string]any{"matchLabels": map[string]string{workerLabel: "isolated-v1"}}, "policyTypes": []string{"Ingress", "Egress"}, "ingress": []any{}, "egress": []any{
			map[string]any{"to": []any{selector(p.config.ExchangePeer)}, "ports": []any{map[string]any{"protocol": "TCP", "port": port}}},
			map[string]any{"to": []any{selector(p.config.DNSPeer)}, "ports": []any{map[string]any{"protocol": "TCP", "port": 53}, map[string]any{"protocol": "UDP", "port": 53}}},
		},
	}
}

func (p *Provider) checkIsolation(ctx context.Context) error {
	var namespace struct {
		Metadata metadata `json:"metadata"`
	}
	if err := p.request(ctx, http.MethodGet, "/api/v1/namespaces/"+p.config.Namespace, nil, nil, &namespace); err != nil || namespace.Metadata.Name != p.config.Namespace || namespace.Metadata.Labels["pod-security.kubernetes.io/enforce"] != "restricted" {
		return hostedexecution.ErrConfiguration
	}
	if err := p.checkAccount(ctx, p.config.ServiceAccount); err != nil {
		return err
	}
	if p.config.Gateway != nil {
		if err := p.checkAccount(ctx, p.config.Gateway.ServiceAccount); err != nil {
			return err
		}
	}
	var runtime struct {
		Metadata   metadata `json:"metadata"`
		Handler    string   `json:"handler"`
		Scheduling struct {
			NodeSelector map[string]string `json:"nodeSelector"`
			Tolerations  []any             `json:"tolerations"`
		} `json:"scheduling"`
	}
	if err := p.request(ctx, http.MethodGet, "/apis/node.k8s.io/v1/runtimeclasses/"+p.config.RuntimeClass, nil, nil, &runtime); err != nil || runtime.Metadata.Name != p.config.RuntimeClass || runtime.Handler == "" || len(runtime.Scheduling.Tolerations) > 0 {
		return hostedexecution.ErrConfiguration
	}
	for key, value := range runtime.Scheduling.NodeSelector {
		want := p.config.NodeSelector[key]
		if key == "kubernetes.io/os" {
			want = "linux"
		}
		if key == "kubernetes.io/arch" {
			want = p.config.Architecture
		}
		if want != value {
			return hostedexecution.ErrConfiguration
		}
	}
	return p.checkPolicies(ctx)
}

func (p *Provider) checkAccount(ctx context.Context, name string) error {
	var account struct {
		Metadata  metadata `json:"metadata"`
		Automount *bool    `json:"automountServiceAccountToken"`
	}
	if err := p.request(ctx, http.MethodGet, "/api/v1/namespaces/"+p.config.Namespace+"/serviceaccounts/"+name, nil, nil, &account); err != nil || account.Metadata.Name != name || account.Automount == nil || *account.Automount {
		return hostedexecution.ErrConfiguration
	}
	for key := range account.Metadata.Annotations {
		if !strings.HasPrefix(key, "meta.helm.sh/") && key != "kubectl.kubernetes.io/last-applied-configuration" {
			return hostedexecution.ErrConfiguration
		}
	}
	return nil
}
