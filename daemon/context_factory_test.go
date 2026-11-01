package daemon

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter/certificate"
	"github.com/sagernet/sing-box/adapter/endpoint"
	"github.com/sagernet/sing-box/adapter/inbound"
	"github.com/sagernet/sing-box/adapter/outbound"
	boxservice "github.com/sagernet/sing-box/adapter/service"
	"github.com/sagernet/sing-box/dns"
	"github.com/sagernet/sing-box/dns/transport/local"
	"github.com/sagernet/sing-box/protocol/direct"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/filemanager"
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

func TestSuccessfulReloadAndCheckDoNotReuseRunningGeneration(t *testing.T) {
	dir := t.TempDir()
	parent := service.ContextWithDefaultRegistry(filemanager.WithDefault(context.Background(), dir, dir, os.Getuid(), os.Getgid()))
	var contexts []context.Context
	s := NewStartedService(ServiceOptions{Context: parent, LogMaxLines: 10, ContextFactory: func(ctx context.Context, check bool) (context.Context, error) {
		if service.FromContext[*factoryMarker](ctx) != nil {
			t.Fatal("inherited live generation")
		}
		service.MustRegister[*factoryMarker](ctx, &factoryMarker{len(contexts) + 1})
		contexts = append(contexts, ctx)
		out := outbound.NewRegistry()
		direct.RegisterOutbound(out)
		transports := dns.NewTransportRegistry()
		local.RegisterTransport(transports)
		return box.Context(ctx, inbound.NewRegistry(), out, endpoint.NewRegistry(), transports, boxservice.NewRegistry(), certificate.NewRegistry()), nil
	}})
	defer s.Close()
	defer s.CloseService()
	const config = `{"log":{"disabled":true},"outbounds":[{"type":"direct","tag":"direct"}]}`
	if err := s.StartOrReloadService(parent, config, nil); err != nil {
		t.Fatal(err)
	}
	first := s.Instance()
	if first == nil || contexts[0].Err() != nil {
		t.Fatal("first run missing")
	}
	if err := s.StartOrReloadService(parent, config, nil); err != nil {
		t.Fatal(err)
	}
	second := s.Instance()
	if second == nil || second == first || contexts[0].Err() != context.Canceled || contexts[1].Err() != nil {
		t.Fatal("reload reused running generation")
	}
	before := s.serviceStatus
	if err := s.CheckConfig(parent, config); err != nil {
		t.Fatal(err)
	}
	if s.Instance() != second || s.serviceStatus != before || contexts[1].Err() != nil || contexts[2].Err() != context.Canceled {
		t.Fatal("check affected live instance")
	}
	if service.FromContext[*factoryMarker](parent) != nil {
		t.Fatal("parent mutated")
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
