package dsl

import (
	"fmt"

	"github.com/alm/domain"
)

// ValidationError describes a single cross-model inconsistency.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate checks that a DeploymentEnv is consistent with its AppArchitecture
// and the pipelines referenced by the architecture's services.
// It returns a slice of all validation errors found (never stops at the first one).
func Validate(
	env *domain.DeploymentEnv,
	arch *domain.AppArchitecture,
	pipelines map[string]*domain.Pipeline,
) []error {
	var errs []error

	add := func(field, msg string, args ...interface{}) {
		errs = append(errs, &ValidationError{
			Field:   field,
			Message: fmt.Sprintf(msg, args...),
		})
	}

	if env.App != arch.Name {
		add("app", "deployment env references app %q but architecture name is %q",
			env.App, arch.Name)
		return errs // remaining checks need the right arch
	}

	// Validate each service listed in the deployment env.
	for _, svcDeploy := range env.Services {
		archSvc := arch.FindService(svcDeploy.Name)
		if archSvc == nil {
			add(fmt.Sprintf("services[%s]", svcDeploy.Name),
				"service %q not found in app architecture %q", svcDeploy.Name, arch.Name)
			continue
		}

		pipeline, ok := pipelines[archSvc.Pipeline]
		if !ok {
			add(fmt.Sprintf("services[%s].pipeline", svcDeploy.Name),
				"pipeline %q not found (referenced by service %q)", archSvc.Pipeline, svcDeploy.Name)
			continue
		}

		if !pipeline.CanProduce(svcDeploy.Accepts) {
			add(fmt.Sprintf("services[%s].accepts", svcDeploy.Name),
				"service %q accepts %q but pipeline %q can only produce %v",
				svcDeploy.Name, svcDeploy.Accepts, pipeline.Name, pipeline.Deliverables)
		}
	}

	// Validate bindings reference known services in the architecture.
	for _, b := range env.Bindings {
		if arch.FindService(b.Service) == nil {
			add(fmt.Sprintf("bindings[%s]", b.Service),
				"binding references unknown service %q", b.Service)
		}
	}

	// Validate network routes reference known services.
	if env.Network != nil {
		for _, ingress := range env.Network.Ingress {
			for _, route := range ingress.Routes {
				if arch.FindService(route.Service) == nil {
					add(fmt.Sprintf("network.ingress[%s].routes[%s]", ingress.Name, route.Path),
						"route references unknown service %q", route.Service)
				}
			}
		}
	}

	return errs
}
