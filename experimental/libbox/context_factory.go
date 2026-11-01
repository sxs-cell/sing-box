package libbox

import (
	"context"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/daemon"
	"github.com/sagernet/sing/service"
)

// This optional Go-only interface is deliberately unexported: context.Context
// must not become part of the gomobile interface implemented by Java/Swift.
// A Go platform adapter may implement it without changing PlatformInterface.
type serviceContextFactory interface {
	NewServiceContext(parent context.Context, forCheck bool) (context.Context, error)
}

func commandContextFactory(platform PlatformInterface) (daemon.ContextFactory, error) {
	factory, ok := platform.(serviceContextFactory)
	if !ok {
		return nil, requireContextFactory()
	}
	return func(parent context.Context, forCheck bool) (context.Context, error) {
		ctx, err := factory.NewServiceContext(parent, forCheck)
		if err != nil || ctx == nil {
			return ctx, err // daemon rejects nil; never select the default registries.
		}
		if forCheck {
			// box.New calls PlatformInterface.Initialize even without Start. Never
			// replace the running platform wrapper's network manager during checks.
			ctx = service.ContextWith[adapter.PlatformInterface](ctx, (*platformInterfaceStub)(nil))
		}
		return ctx, nil
	}, nil
}
