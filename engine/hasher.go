package engine

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/alm/domain"
)

// hashableServiceConfig is a stable representation of the config fields that
// matter for determining whether a service needs rebuilding / redeployment.
type hashableServiceConfig struct {
	Pipeline   string `json:"pipeline"`
	Repository string `json:"repository"`
	Accepts    string `json:"accepts"`
	Compute    string `json:"compute"` // JSON of ComputeSpec
}

// HashServiceConfig computes a deterministic hash over the union of
// ServiceSpec, ServiceDeploySpec, and Pipeline fields that would trigger
// a rebuild or redeploy when changed.
func HashServiceConfig(svc *domain.ServiceSpec, deploy *domain.ServiceDeploySpec, pipeline *domain.Pipeline) string {
	computeJSON, _ := json.Marshal(deploy.Compute)
	h := hashableServiceConfig{
		Pipeline:   svc.Pipeline,
		Repository: svc.Repository,
		Accepts:    deploy.Accepts,
		Compute:    string(computeJSON),
	}
	return jsonSHA256(h)
}

// hashableInfraConfig captures the fields of an InfraResource that affect its
// provisioned state.
type hashableInfraConfig struct {
	Type      string `json:"type"`
	Provision string `json:"provision"` // JSON of InfraProvision
	Resources string `json:"resources"` // JSON of ResourceSpec
	Config    string `json:"config"`    // JSON of Config map
}

// HashInfraConfig computes a deterministic hash for an infrastructure resource.
func HashInfraConfig(dep *domain.InfraResource) string {
	provJSON, _ := json.Marshal(dep.Provision)
	resJSON, _ := json.Marshal(dep.Resources)
	cfgJSON, _ := marshalSortedMap(dep.Config)
	h := hashableInfraConfig{
		Type:      dep.Type,
		Provision: string(provJSON),
		Resources: string(resJSON),
		Config:    string(cfgJSON),
	}
	return jsonSHA256(h)
}

// hashableIngressConfig captures the fields of an IngressSpec that affect its
// configured state.
type hashableIngressConfig struct {
	Type   string `json:"type"`
	Bind   string `json:"bind"`
	TLS    string `json:"tls"`
	Routes string `json:"routes"`
}

// HashIngressConfig computes a deterministic hash for an ingress resource.
func HashIngressConfig(ing *domain.IngressSpec) string {
	bindJSON, _ := json.Marshal(ing.Bind)
	tlsJSON, _ := json.Marshal(ing.TLS)
	routesJSON, _ := json.Marshal(ing.Routes)
	h := hashableIngressConfig{
		Type:   ing.Type,
		Bind:   string(bindJSON),
		TLS:    string(tlsJSON),
		Routes: string(routesJSON),
	}
	return jsonSHA256(h)
}

func jsonSHA256(v interface{}) string {
	data, _ := json.Marshal(v)
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

func marshalSortedMap(m map[string]interface{}) ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sorted := make([]interface{}, 0, len(keys)*2)
	for _, k := range keys {
		sorted = append(sorted, k, m[k])
	}
	return json.Marshal(sorted)
}
