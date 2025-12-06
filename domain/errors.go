package domain

import "errors"

var (
	ErrInvalidStateMachineTemplate = errors.New("invalid state machine template")
	ErrEmptyStates                 = errors.New("state machine template has no states")
	ErrInvalidTransition           = errors.New("invalid state transition")
	ErrTransitionRuleNotFound      = errors.New("transition rule not found")
	ErrActionMismatch              = errors.New("action mismatch")
)

