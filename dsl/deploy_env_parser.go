package dsl

import (
	"fmt"
	"os"

	"github.com/alm/domain"
	"gopkg.in/yaml.v3"
)

// deployEnvDSL mirrors the DeploymentEnv YAML structure.
type deployEnvDSL struct {
	Kind         string             `yaml:"kind"`
	Name         string             `yaml:"name"`
	Environment  string             `yaml:"environment"`
	App          string             `yaml:"app"`
	Services     []serviceDeployDSL `yaml:"services"`
	Dependencies []infraResDSL      `yaml:"dependencies"`
	Bindings     []bindingDSL       `yaml:"bindings"`
	Network      *networkDSL        `yaml:"network"`
}

type serviceDeployDSL struct {
	Name    string      `yaml:"name"`
	Accepts string      `yaml:"accepts"`
	Compute *computeDSL `yaml:"compute"`
}

type computeDSL struct {
	Type      string       `yaml:"type"`
	Resources *resourceDSL `yaml:"resources"`
	Volumes   []volumeDSL  `yaml:"volumes"`
	Ports     []int        `yaml:"ports"`
}

type resourceDSL struct {
	CPU      string `yaml:"cpu"`
	Memory   string `yaml:"memory"`
	Storage  string `yaml:"storage"`
	Replicas int    `yaml:"replicas"`
}

type volumeDSL struct {
	Name  string `yaml:"name"`
	Size  string `yaml:"size"`
	Mount string `yaml:"mount"`
}

type infraResDSL struct {
	Name      string                 `yaml:"name"`
	Type      string                 `yaml:"type"`
	Provision *provisionDSL          `yaml:"provision"`
	Resources *resourceDSL           `yaml:"resources"`
	Config    map[string]interface{} `yaml:"config"`
}

type provisionDSL struct {
	Via      string                 `yaml:"via"`
	Image    string                 `yaml:"image"`
	Env      map[string]string      `yaml:"env"`
	Module   string                 `yaml:"module"`
	Vars     map[string]interface{} `yaml:"vars"`
	Chart    string                 `yaml:"chart"`
	Values   map[string]interface{} `yaml:"values"`
	Endpoint string                 `yaml:"endpoint"`
}

type bindingDSL struct {
	Service string            `yaml:"service"`
	Env     map[string]string `yaml:"env"`
}

type networkDSL struct {
	Ingress []ingressDSL `yaml:"ingress"`
}

type ingressDSL struct {
	Name      string       `yaml:"name"`
	Type      string       `yaml:"type"`
	Bind      *bindDSL     `yaml:"bind"`
	TLS       *tlsDSL      `yaml:"tls"`
	Routes    []routeDSL   `yaml:"routes"`
	Resources *resourceDSL `yaml:"resources"`
}

type bindDSL struct {
	IP    string `yaml:"ip"`
	HTTP  int    `yaml:"http"`
	HTTPS int    `yaml:"https"`
}

type tlsDSL struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

type routeDSL struct {
	Path    string `yaml:"path"`
	Service string `yaml:"service"`
	Port    int    `yaml:"port"`
}

// ParseDeploymentEnv reads a DeploymentEnv YAML file and returns a domain.DeploymentEnv.
func ParseDeploymentEnv(path string) (*domain.DeploymentEnv, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading deployment env file %s: %w", path, err)
	}
	return parseDeployEnvBytes(data, path)
}

func parseDeployEnvBytes(data []byte, source string) (*domain.DeploymentEnv, error) {
	var d deployEnvDSL
	if err := yaml.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("parsing deployment env YAML %s: %w", source, err)
	}

	if d.Kind != "DeploymentEnv" {
		return nil, fmt.Errorf("%s: expected kind DeploymentEnv, got %q", source, d.Kind)
	}
	if d.Name == "" {
		return nil, fmt.Errorf("%s: deployment env name is required", source)
	}
	if d.App == "" {
		return nil, fmt.Errorf("%s: deployment env must reference an app (app field)", source)
	}

	env := &domain.DeploymentEnv{
		Name:        d.Name,
		Environment: d.Environment,
		App:         d.App,
	}

	for i, s := range d.Services {
		if s.Name == "" {
			return nil, fmt.Errorf("%s: services[%d] name is required", source, i)
		}
		if s.Accepts == "" {
			return nil, fmt.Errorf("%s: service %q must declare which asset type it accepts", source, s.Name)
		}
		env.Services = append(env.Services, &domain.ServiceDeploySpec{
			Name:    s.Name,
			Accepts: s.Accepts,
			Compute: toComputeSpec(s.Compute),
		})
	}

	for i, dep := range d.Dependencies {
		if dep.Name == "" {
			return nil, fmt.Errorf("%s: dependencies[%d] name is required", source, i)
		}
		if dep.Type == "" {
			return nil, fmt.Errorf("%s: dependency %q must declare its type", source, dep.Name)
		}
		provision, err := toInfraProvision(dep.Provision, dep.Name, source)
		if err != nil {
			return nil, err
		}
		env.Dependencies = append(env.Dependencies, &domain.InfraResource{
			Name:      dep.Name,
			Type:      dep.Type,
			Provision: provision,
			Resources: toResourceSpec(dep.Resources),
			Config:    dep.Config,
		})
	}

	for _, b := range d.Bindings {
		env.Bindings = append(env.Bindings, &domain.Binding{
			Service: b.Service,
			Env:     b.Env,
		})
	}

	if d.Network != nil {
		env.Network = &domain.NetworkConfig{}
		for _, ing := range d.Network.Ingress {
			spec := &domain.IngressSpec{
				Name:      ing.Name,
				Type:      ing.Type,
				Resources: toResourceSpec(ing.Resources),
			}
			if ing.Bind != nil {
				spec.Bind = &domain.BindSpec{
					IP:    ing.Bind.IP,
					HTTP:  ing.Bind.HTTP,
					HTTPS: ing.Bind.HTTPS,
				}
			}
			if ing.TLS != nil {
				spec.TLS = &domain.TLSSpec{Cert: ing.TLS.Cert, Key: ing.TLS.Key}
			}
			for _, r := range ing.Routes {
				spec.Routes = append(spec.Routes, &domain.RouteSpec{
					Path:    r.Path,
					Service: r.Service,
					Port:    r.Port,
				})
			}
			env.Network.Ingress = append(env.Network.Ingress, spec)
		}
	}

	return env, nil
}

// ---- helpers ---------------------------------------------------------------

func toComputeSpec(c *computeDSL) *domain.ComputeSpec {
	if c == nil {
		return nil
	}
	spec := &domain.ComputeSpec{
		Type:      c.Type,
		Resources: toResourceSpec(c.Resources),
		Ports:     c.Ports,
	}
	for _, v := range c.Volumes {
		spec.Volumes = append(spec.Volumes, &domain.VolumeSpec{
			Name:  v.Name,
			Size:  v.Size,
			Mount: v.Mount,
		})
	}
	return spec
}

func toInfraProvision(p *provisionDSL, name, source string) (*domain.InfraProvision, error) {
	if p == nil {
		return nil, nil
	}
	via := domain.ProvisionVia(p.Via)
	switch via {
	case domain.ProvisionViaDocker:
		if p.Image == "" {
			return nil, fmt.Errorf("%s: dependency %q provision via docker requires an image", source, name)
		}
	case domain.ProvisionViaTerraform:
		if p.Module == "" {
			return nil, fmt.Errorf("%s: dependency %q provision via terraform requires a module path", source, name)
		}
	case domain.ProvisionViaHelm:
		if p.Chart == "" {
			return nil, fmt.Errorf("%s: dependency %q provision via helm requires a chart", source, name)
		}
	case domain.ProvisionViaExternal:
		if p.Endpoint == "" {
			return nil, fmt.Errorf("%s: dependency %q provision via external requires an endpoint", source, name)
		}
	default:
		return nil, fmt.Errorf("%s: dependency %q has unknown provision via %q (allowed: docker, terraform, helm, external)",
			source, name, p.Via)
	}
	return &domain.InfraProvision{
		Via:      via,
		Image:    p.Image,
		Env:      p.Env,
		Module:   p.Module,
		Vars:     p.Vars,
		Chart:    p.Chart,
		Values:   p.Values,
		Endpoint: p.Endpoint,
	}, nil
}

func toResourceSpec(r *resourceDSL) *domain.ResourceSpec {
	if r == nil {
		return nil
	}
	return &domain.ResourceSpec{
		CPU:      r.CPU,
		Memory:   r.Memory,
		Storage:  r.Storage,
		Replicas: r.Replicas,
	}
}
