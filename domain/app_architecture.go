package domain

// AppArchitecture defines the logical structure of an application.
// It describes which services compose the application and their dependencies.
// This is the app developer's concern: what services exist and how they relate.
type AppArchitecture struct {
	Name        string
	Description string
	Services    []*ServiceSpec
}

// ServiceSpec defines a single service within the application architecture.
type ServiceSpec struct {
	Name       string
	Pipeline   string   // name of the pipeline template to use for building this service
	Repository string   // git repository URL
	DependsOn  []string // names of services this service depends on (must be deployed first)
}

// TopologicalOrder returns services sorted so that each service appears after
// all of its dependencies. Returns ErrCyclicDependency if a cycle exists.
func (a *AppArchitecture) TopologicalOrder() ([]*ServiceSpec, error) {
	inDegree := make(map[string]int, len(a.Services))
	dependents := make(map[string][]string) // service → services that depend on it
	byName := make(map[string]*ServiceSpec, len(a.Services))

	for _, s := range a.Services {
		byName[s.Name] = s
		if _, ok := inDegree[s.Name]; !ok {
			inDegree[s.Name] = 0
		}
		for _, dep := range s.DependsOn {
			dependents[dep] = append(dependents[dep], s.Name)
			inDegree[s.Name]++
		}
	}

	// Start with services that have no dependencies.
	queue := make([]string, 0)
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	result := make([]*ServiceSpec, 0, len(a.Services))
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		result = append(result, byName[curr])
		for _, next := range dependents[curr] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(result) != len(a.Services) {
		return nil, ErrCyclicDependency
	}
	return result, nil
}

// FindService returns the ServiceSpec with the given name, or nil if not found.
func (a *AppArchitecture) FindService(name string) *ServiceSpec {
	for _, s := range a.Services {
		if s.Name == name {
			return s
		}
	}
	return nil
}
