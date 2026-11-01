package libbox

import (
	"context"
	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/dns"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/service/filemanager"
)

func baseContext(platformInterface PlatformInterface) context.Context {
	dnsRegistry := include.DNSTransportRegistry()
	if platformInterface != nil {
		if localTransport := platformInterface.LocalDNSTransport(); localTransport != nil {
			dns.RegisterTransport[option.LocalDNSServerOptions](dnsRegistry, C.DNSTypeLocal, func(ctx context.Context, logger log.ContextLogger, tag string, options option.LocalDNSServerOptions) (adapter.DNSTransport, error) {
				return newPlatformTransport(ctx, logger, localTransport, tag, options)
			})
		}
	}
	ctx := context.Background()
	ctx = filemanager.WithDefault(ctx, sWorkingPath, sTempPath, sUserID, sGroupID)
	return box.Context(ctx, include.InboundRegistry(), include.OutboundRegistry(), include.EndpointRegistry(), dnsRegistry, include.ServiceRegistry(), include.CertificateProviderRegistry())
}

func standaloneContext() (context.Context, error) { return baseContext(nil), nil }

func requireContextFactory() error { return nil }
