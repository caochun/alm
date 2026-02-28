package api

import (
	"fmt"
	"regexp"

	"github.com/alm/domain"
)

// Node represents a graph node with position and data
type Node struct {
	ID   string         `json:"id"`
	Type string         `json:"type"`
	X    float64        `json:"x"`
	Y    float64        `json:"y"`
	Data map[string]any `json:"data"`
}

// Edge represents a directed edge between two nodes
type Edge struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
}

// GraphData is the top-level response for the graph API
type GraphData struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// column x positions (left edge of node)
const (
	colService     = 30.0
	colDeliverable = 300.0
	colCompute     = 570.0
	colInfra       = 840.0

	nodeWidth       = 240.0
	rowHeightSvc    = 160.0
	rowHeightInfra  = 140.0
	startY          = 60.0
)

var bindingRefRe = regexp.MustCompile(`\$\{(\w[\w-]*)\.[^}]+\}`)

// BuildGraph constructs GraphData from the three DSL models.
// pipelines may be nil (only used to look up pipeline names).
func BuildGraph(
	arch *domain.AppArchitecture,
	env *domain.DeploymentEnv,
	pipelines map[string]*domain.Pipeline,
) GraphData {
	var nodes []Node
	var edges []Edge

	// ----- topological order for column-1 y positioning -----
	ordered, err := arch.TopologicalOrder()
	if err != nil {
		// fallback: original order
		ordered = arch.Services
	}

	// map service name → y position (col 1 & 2 & 3 share same y)
	svcY := make(map[string]float64, len(ordered))
	for i, svc := range ordered {
		svcY[svc.Name] = startY + float64(i)*rowHeightSvc
	}

	// ----- column 1: Service nodes + depends_on edges -----
	for _, svc := range ordered {
		y := svcY[svc.Name]
		nodes = append(nodes, Node{
			ID:   "svc-" + svc.Name,
			Type: "service",
			X:    colService,
			Y:    y,
			Data: map[string]any{
				"name":       svc.Name,
				"pipeline":   svc.Pipeline,
				"repository": svc.Repository,
			},
		})
		for _, dep := range svc.DependsOn {
			edges = append(edges, Edge{
				ID:     fmt.Sprintf("dep-%s-%s", dep, svc.Name),
				Type:   "dependsOn",
				Source: "svc-" + dep,
				Target: "svc-" + svc.Name,
			})
		}
	}

	// ----- column 2: Deliverable nodes + pipeline edges -----
	// ----- column 3: Compute nodes + provision edges -----
	for _, svc := range ordered {
		y := svcY[svc.Name]

		// find deploy spec for this service
		deploySpec := env.FindServiceSpec(svc.Name)
		artifactType := ""
		if deploySpec != nil {
			artifactType = deploySpec.Accepts
		}

		// pipeline name for label
		pipelineName := svc.Pipeline

		// deliverable node
		nodes = append(nodes, Node{
			ID:   "del-" + svc.Name,
			Type: "deliverable",
			X:    colDeliverable,
			Y:    y,
			Data: map[string]any{
				"service":      svc.Name,
				"artifactType": artifactType,
				"pipeline":     pipelineName,
			},
		})

		// pipeline edge: service → deliverable
		edges = append(edges, Edge{
			ID:     "pipe-" + svc.Name,
			Type:   "pipeline",
			Source: "svc-" + svc.Name,
			Target: "del-" + svc.Name,
			Label:  pipelineName,
		})

		// compute node
		if deploySpec != nil {
			computeData := map[string]any{
				"name":        svc.Name,
				"computeType": "",
				"accepts":     artifactType,
				"cpu":         "",
				"memory":      "",
				"replicas":    1,
			}
			if deploySpec.Compute != nil {
				computeData["computeType"] = deploySpec.Compute.Type
				if deploySpec.Compute.Resources != nil {
					computeData["cpu"] = deploySpec.Compute.Resources.CPU
					computeData["memory"] = deploySpec.Compute.Resources.Memory
					if deploySpec.Compute.Resources.Replicas > 0 {
						computeData["replicas"] = deploySpec.Compute.Resources.Replicas
					}
				}
			}
			nodes = append(nodes, Node{
				ID:   "cmp-" + svc.Name,
				Type: "compute",
				X:    colCompute,
				Y:    y,
				Data: computeData,
			})

			// provision edge: deliverable → compute
			// via label: derive from compute type
			via := computeTypeToVia(deploySpec.Compute)
			edges = append(edges, Edge{
				ID:     "prov-" + svc.Name,
				Type:   "provision",
				Source: "del-" + svc.Name,
				Target: "cmp-" + svc.Name,
				Label:  via,
			})
		}
	}

	// ----- column 4: Infra nodes -----
	infraY := make(map[string]float64, len(env.Dependencies))
	for i, dep := range env.Dependencies {
		y := startY + float64(i)*rowHeightInfra
		infraY[dep.Name] = y

		via := ""
		if dep.Provision != nil {
			via = string(dep.Provision.Via)
		}
		nodes = append(nodes, Node{
			ID:   "inf-" + dep.Name,
			Type: "infra",
			X:    colInfra,
			Y:    y,
			Data: map[string]any{
				"name":         dep.Name,
				"resourceType": dep.Type,
				"via":          via,
			},
		})
	}

	// ----- column 4: Ingress nodes (after infra) -----
	ingressStartY := startY + float64(len(env.Dependencies))*rowHeightInfra
	if len(env.Dependencies) > 0 {
		ingressStartY += 20 // extra gap
	}
	ingressY := make(map[string]float64)
	if env.Network != nil {
		for i, ing := range env.Network.Ingress {
			y := ingressStartY + float64(i)*rowHeightInfra

			ingressY[ing.Name] = y

			routes := make([]map[string]any, 0, len(ing.Routes))
			for _, r := range ing.Routes {
				routes = append(routes, map[string]any{
					"path":    r.Path,
					"service": r.Service,
					"port":    r.Port,
				})
			}

			bindData := map[string]any{}
			if ing.Bind != nil {
				bindData["http"] = ing.Bind.HTTP
				bindData["https"] = ing.Bind.HTTPS
				bindData["ip"] = ing.Bind.IP
			}

			nodes = append(nodes, Node{
				ID:   "ing-" + ing.Name,
				Type: "ingress",
				X:    colInfra,
				Y:    y,
				Data: map[string]any{
					"name":        ing.Name,
					"ingressType": ing.Type,
					"bind":        bindData,
					"routes":      routes,
				},
			})
		}
	}

	// ----- binding edges: compute → infra -----
	// parse ${infraName.field} references from bindings
	for _, binding := range env.Bindings {
		referencedInfra := extractInfraRefs(binding.Env)
		for infraName := range referencedInfra {
			edgeID := fmt.Sprintf("bind-%s-%s", binding.Service, infraName)
			edges = append(edges, Edge{
				ID:     edgeID,
				Type:   "binding",
				Source: "cmp-" + binding.Service,
				Target: "inf-" + infraName,
			})
		}
	}

	// ----- route edges: ingress → compute -----
	if env.Network != nil {
		for _, ing := range env.Network.Ingress {
			seenRoutes := make(map[string]bool)
			for _, route := range ing.Routes {
				key := ing.Name + "-" + route.Service
				if seenRoutes[key] {
					continue
				}
				seenRoutes[key] = true
				edges = append(edges, Edge{
					ID:     fmt.Sprintf("route-%s-%s", ing.Name, route.Service),
					Type:   "route",
					Source: "ing-" + ing.Name,
					Target: "cmp-" + route.Service,
					Label:  route.Path,
				})
			}
		}
	}

	return GraphData{Nodes: nodes, Edges: edges}
}

// computeTypeToVia maps compute type string to a via label for the provision edge.
func computeTypeToVia(compute *domain.ComputeSpec) string {
	if compute == nil {
		return ""
	}
	switch compute.Type {
	case "docker-container":
		return "docker"
	case "kubernetes-pod":
		return "k8s"
	case "nginx-static":
		return "nginx"
	case "vm":
		return "vm"
	default:
		return compute.Type
	}
}

// extractInfraRefs parses env-value templates and returns unique infra resource names referenced.
func extractInfraRefs(envMap map[string]string) map[string]bool {
	refs := make(map[string]bool)
	for _, val := range envMap {
		matches := bindingRefRe.FindAllStringSubmatch(val, -1)
		for _, m := range matches {
			if len(m) >= 2 {
				refs[m[1]] = true
			}
		}
	}
	return refs
}
