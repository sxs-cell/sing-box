package daemon

import (
	"context"
	"errors"
)

// ContextFactory must derive fresh registries and run state from parent. The
// parent already has an independent service registry and lifecycle cancellation.
// forCheck must not publish a running generation or perform platform operations.
// Returning an error (or a nil context) fails the operation without fallback.
type ContextFactory func(parent context.Context, forCheck bool) (context.Context, error)

func (s *StartedService) prepareContext(parent context.Context, forCheck bool) (context.Context, error) {
	if s.contextFactory == nil {
		return parent, nil
	}
	ctx, err := s.contextFactory(parent, forCheck)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		return nil, errors.New("service context factory returned nil context")
	}
	return ctx, nil
}
