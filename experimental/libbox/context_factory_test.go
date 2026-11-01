package libbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/certificate"
	"github.com/sagernet/sing-box/adapter/endpoint"
	"github.com/sagernet/sing-box/adapter/inbound"
	"github.com/sagernet/sing-box/adapter/outbound"
	boxservice "github.com/sagernet/sing-box/adapter/service"
	"github.com/sagernet/sing-box/dns"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/protocol/direct"
	"github.com/sagernet/sing/service"
)

type testPlatform struct{ PlatformInterface }

func (*testPlatform) UseProcFS() bool                      { return false }
func (*testPlatform) LocalDNSTransport() LocalDNSTransport { return nil }

type factoryPlatform struct {
	testPlatform
	factory func(context.Context, bool) (context.Context, error)
}

func (p *factoryPlatform) NewServiceContext(ctx context.Context, check bool) (context.Context, error) {
	return p.factory(ctx, check)
}

func testPaths(t *testing.T) string {
	t.Helper()
	base, working, temp, lines := sBasePath, sWorkingPath, sTempPath, sLogMaxLines
	dir := t.TempDir()
	sBasePath, sWorkingPath, sTempPath, sLogMaxLines = dir, dir, dir, 100
	t.Cleanup(func() { sBasePath, sWorkingPath, sTempPath, sLogMaxLines = base, working, temp, lines })
	return dir
}

func TestCommandFactoryIsLazyAndErrorsReachRunAndCheck(t *testing.T) {
	testPaths(t)
	want := errors.New("caller registry unavailable")
	calls := 0
	p := &factoryPlatform{factory: func(context.Context, bool) (context.Context, error) { calls++; return nil, want }}
	s, err := NewCommandServer(nil, p)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if calls != 0 {
		t.Fatal("eager factory")
	}
	if err = s.StartedService.CheckConfig(s.ctx, "{}"); err != want {
		t.Fatalf("check error changed: %v", err)
	}
	if err = s.StartOrReloadService("{}", &OverrideOptions{}); !errors.Is(err, want) {
		t.Fatalf("run error changed: %v", err)
	}
	if calls != 2 || s.StartedService.Instance() != nil {
		t.Fatal("factory bypassed")
	}
}

func TestCommandCheckParsesCallerOutboundWithoutTouchingLivePlatformOrFiles(t *testing.T) {
	dir := testPaths(t)
	var checks []bool
	p := &factoryPlatform{factory: func(ctx context.Context, check bool) (context.Context, error) {
		checks = append(checks, check)
		out := outbound.NewRegistry()
		outbound.Register[option.DirectOutboundOptions](out, "factory-only", direct.NewOutbound)
		ctx = box.Context(ctx, inbound.NewRegistry(), out, endpoint.NewRegistry(), dns.NewTransportRegistry(), boxservice.NewRegistry(), certificate.NewRegistry())
		ctx = service.ContextWith[option.OutboundOptionsRegistry](ctx, out)
		ctx = service.ContextWith[adapter.OutboundRegistry](ctx, out)
		return ctx, nil
	}}
	s, err := NewCommandServer(nil, p)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	live := s.platformWrapper
	for range 2 {
		if err = s.StartedService.CheckConfig(s.ctx, `{"log":{"output":"check.log"},"outbounds":[{"type":"factory-only","tag":"test"}],"experimental":{"cache_file":{"enabled":true,"path":"check.db"}}}`); err != nil {
			t.Fatal(err)
		}
	}
	if len(checks) != 2 || !checks[0] || !checks[1] || live.networkManager != nil || s.StartedService.Instance() != nil {
		t.Fatal("validation changed live generation or platform")
	}
	if service.FromContext[adapter.OutboundRegistry](s.ctx) != nil {
		// The default build starts with default registries; the caller-only
		// outbound must never be added to that parent registry.
		if _, ok := service.FromContext[option.OutboundOptionsRegistry](s.ctx).CreateOptions("factory-only"); ok {
			t.Fatal("parent registry changed")
		}
	}
	for _, name := range []string{"check.log", "check.db", "configuration.json"} {
		if _, err = os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("check wrote file", name, err)
		}
	}
}

// Keep the platform-validation guard structural as well as behavioral.
var _ adapter.PlatformInterface = (*platformInterfaceStub)(nil)
