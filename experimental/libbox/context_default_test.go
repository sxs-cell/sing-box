package libbox

import (
	"testing"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/service"
)

func TestDefaultRegistriesAndConfigEntrypointsRemainAvailable(t *testing.T) {
	testPaths(t)
	factory, err := commandContextFactory(&testPlatform{})
	if err != nil || factory != nil {
		t.Fatal("optional factory became mandatory", err)
	}
	ctx := baseContext(nil)
	registry := service.FromContext[option.OutboundOptionsRegistry](ctx)
	for _, kind := range []string{"direct", "tor", "vless", "urltest"} {
		if _, ok := registry.CreateOptions(kind); !ok {
			t.Fatal("default outbound lost", kind)
		}
	}
	const config = `{"outbounds":[{"type":"direct","tag":"direct"}]}`
	if err = CheckConfig(config); err != nil {
		t.Fatal(err)
	}
	if _, err = FormatConfig(config); err != nil {
		t.Fatal(err)
	}
	if tun, err := HasTunInbound(config); err != nil || tun {
		t.Fatal(tun, err)
	}
	s, err := NewCommandServer(nil, &testPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.CheckConfig(config); err != nil {
		t.Fatal(err)
	}
}
