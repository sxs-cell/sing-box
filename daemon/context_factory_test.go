package daemon

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sagernet/sing/service"
)

type factoryMarker struct{ generation int }

func TestContextFactoryIsOptionalLazyAndFailsWithoutFallback(t *testing.T) {
	parent := service.ContextWithDefaultRegistry(context.Background())
	plain := NewStartedService(ServiceOptions{Context: parent})
	defer plain.Close()
	if ctx, err := plain.prepareContext(parent, false); err != nil || ctx != parent {
		t.Fatal("nil factory changed default context", err)
	}
	want := errors.New("factory failure")
	calls := 0
	s := NewStartedService(ServiceOptions{Context: parent, ContextFactory: func(context.Context, bool) (context.Context, error) {
		calls++
		return nil, want
	}})
	defer s.Close()
	if calls != 0 {
		t.Fatal("factory called during construction")
	}
	if err := s.StartOrReloadService(parent, "{", nil); err != want {
		t.Fatalf("factory error lost or decoding fallback used: %v", err)
	}
	if err := s.CheckConfig(parent, "{"); err != want {
		t.Fatalf("check fallback: %v", err)
	}
	if calls != 2 || s.Instance() != nil {
		t.Fatal("unexpected calls or published instance")
	}
	s.contextFactory = func(context.Context, bool) (context.Context, error) { return nil, nil }
	if err := s.CheckConfig(parent, "{}"); err == nil || !strings.Contains(err.Error(), "nil context") {
		t.Fatalf("nil factory context accepted: %v", err)
	}
}

func TestContextFactoryRunAndCheckHaveIsolatedCanceledGenerations(t *testing.T) {
	parent := service.ContextWithDefaultRegistry(context.Background())
	var contexts []context.Context
	var checks []bool
	s := NewStartedService(ServiceOptions{Context: parent, ContextFactory: func(ctx context.Context, check bool) (context.Context, error) {
		if service.FromContext[*factoryMarker](ctx) != nil {
			t.Fatal("previous generation leaked")
		}
		service.MustRegister[*factoryMarker](ctx, &factoryMarker{len(contexts) + 1})
		contexts = append(contexts, ctx)
		checks = append(checks, check)
		return ctx, nil
	}})
	defer s.Close()
	for range 2 {
		if err := s.StartOrReloadService(parent, "{", nil); err == nil || !strings.Contains(err.Error(), "decode config") {
			t.Fatalf("invalid config: %v", err)
		}
	}
	before := s.serviceStatus
	if err := s.CheckConfig(parent, "{"); err == nil {
		t.Fatal("invalid check accepted")
	}
	if s.serviceStatus != before || s.Instance() != nil {
		t.Fatal("check changed running state")
	}
	if len(contexts) != 3 || checks[0] || checks[1] || !checks[2] {
		t.Fatal(checks)
	}
	for i, ctx := range contexts {
		if ctx.Err() != context.Canceled || service.FromContext[*factoryMarker](ctx).generation != i+1 {
			t.Fatal("generation not isolated/canceled", i)
		}
	}
	if service.FromContext[*factoryMarker](parent) != nil || parent.Err() != nil {
		t.Fatal("parent mutated")
	}
}
