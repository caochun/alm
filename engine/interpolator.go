package engine

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/alm/domain"
)

var interpolateRe = regexp.MustCompile(`\$\{(\w[\w-]*)\.([^}]+)\}`)

// Interpolate resolves ${infraName.field} references in a binding's env map
// by looking up runtime outputs from provisioned infrastructure state.
// Unknown references are returned as errors.
func Interpolate(envMap map[string]string, infraStates map[string]*domain.InfraState) (map[string]string, error) {
	result := make(map[string]string, len(envMap))
	var errs []string

	for key, tmpl := range envMap {
		resolved := interpolateRe.ReplaceAllStringFunc(tmpl, func(match string) string {
			parts := interpolateRe.FindStringSubmatch(match)
			if len(parts) != 3 {
				errs = append(errs, fmt.Sprintf("invalid interpolation: %s", match))
				return match
			}
			infraName := parts[1]
			field := parts[2]

			state, ok := infraStates[infraName]
			if !ok {
				errs = append(errs, fmt.Sprintf("infra %q not found for %s", infraName, match))
				return match
			}
			val, ok := state.Outputs[field]
			if !ok {
				errs = append(errs, fmt.Sprintf("field %q not found in infra %q outputs for %s", field, infraName, match))
				return match
			}
			return val
		})
		result[key] = resolved
	}

	if len(errs) > 0 {
		return result, fmt.Errorf("interpolation errors: %s", strings.Join(errs, "; "))
	}
	return result, nil
}
