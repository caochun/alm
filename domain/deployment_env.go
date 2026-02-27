package domain

// DeploymentEnv defines how an application is deployed in a specific environment.
// This is the infrastructure operator's concern: what compute and supporting
// services are needed, and how application services connect to them.
type DeploymentEnv struct {
	Name         string
	Environment  string             // e.g., development, staging, production
	App          string             // references AppArchitecture.Name
	Services     []*ServiceDeploySpec
	Dependencies []*InfraResource
	Bindings     []*Binding
	Network      *NetworkConfig
}

// ServiceDeploySpec defines the deployment configuration for a single service.
type ServiceDeploySpec struct {
	Name    string      // must match a ServiceSpec.Name in the referenced AppArchitecture
	Accepts string      // asset type this service expects from the pipeline (e.g., docker-image)
	Compute *ComputeSpec
}

// ComputeSpec describes the compute environment that will run the service.
type ComputeSpec struct {
	Type      string       // docker-container | kubernetes-pod | nginx-static | vm
	Resources *ResourceSpec
	Volumes   []*VolumeSpec
	Ports     []int
}

// ProvisionVia identifies the provisioning mechanism for an infrastructure resource.
type ProvisionVia string

const (
	ProvisionViaDocker    ProvisionVia = "docker"    // docker run
	ProvisionViaTerraform ProvisionVia = "terraform" // terraform apply
	ProvisionViaHelm      ProvisionVia = "helm"      // helm install
	ProvisionViaExternal  ProvisionVia = "external"  // pre-existing; no provisioning needed
)

// InfraProvision defines how an infrastructure resource is brought into existence.
// The active fields depend on the Via value.
type InfraProvision struct {
	Via ProvisionVia // docker | terraform | helm | external

	// docker: run a container image
	Image string
	Env   map[string]string // env vars injected into the container at startup

	// terraform: apply a module
	Module string
	Vars   map[string]interface{} // terraform input variables

	// helm: install a chart
	Chart  string
	Values map[string]interface{} // helm values

	// external: resource already exists; supply its reachable endpoint
	Endpoint string
}

// InfraResource defines a supporting infrastructure resource such as a database,
// cache, or message queue that one or more services depend on.
type InfraResource struct {
	Name      string
	Type      string                 // e.g., mysql:8.0, redis:7, kafka:3.6
	Provision *InfraProvision        // how to provision this resource
	Resources *ResourceSpec
	Config    map[string]interface{} // runtime parameters (port, database name, etc.)
}

// Binding connects a service to supporting infrastructure via environment variable
// injection. Value templates may reference infra properties using ${infra.field} syntax.
type Binding struct {
	Service string
	Env     map[string]string // env var name → value template
}

// NetworkConfig defines the network topology for the deployment environment.
type NetworkConfig struct {
	Ingress []*IngressSpec
}

// IngressSpec defines a single ingress entry point (load balancer / reverse proxy).
type IngressSpec struct {
	Name      string
	Type      string       // nginx | traefik | haproxy
	Bind      *BindSpec
	TLS       *TLSSpec
	Routes    []*RouteSpec
	Resources *ResourceSpec
}

// BindSpec defines the IP address and port bindings for an ingress.
type BindSpec struct {
	IP    string
	HTTP  int
	HTTPS int
}

// TLSSpec holds TLS certificate paths.
type TLSSpec struct {
	Cert string
	Key  string
}

// RouteSpec defines a single routing rule: HTTP path prefix → service:port.
type RouteSpec struct {
	Path    string
	Service string
	Port    int
}

// FindDependency returns the InfraResource with the given name, or nil.
func (e *DeploymentEnv) FindDependency(name string) *InfraResource {
	for _, d := range e.Dependencies {
		if d.Name == name {
			return d
		}
	}
	return nil
}

// FindServiceSpec returns the ServiceDeploySpec for the given service name, or nil.
func (e *DeploymentEnv) FindServiceSpec(name string) *ServiceDeploySpec {
	for _, s := range e.Services {
		if s.Name == name {
			return s
		}
	}
	return nil
}
