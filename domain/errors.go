package domain

import "errors"

var (
	ErrCyclicDependency = errors.New("cyclic dependency detected in service graph")
	ErrInvalidKind      = errors.New("invalid DSL kind")
)
