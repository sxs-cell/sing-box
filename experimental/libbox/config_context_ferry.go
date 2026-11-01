//go:build ferry_mobile

package libbox

import (
	"context"
	"errors"

	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/filemanager"
)

var (
	errFerryUnsupported            = errors.New("本构建不支持此功能 (ferry_mobile)")
	errFerryContextFactoryRequired = errors.New("ferry_mobile requires a caller service context factory; default include registries are unavailable")
)

func baseContext(PlatformInterface) context.Context {
	ctx := filemanager.WithDefault(context.Background(), sWorkingPath, sTempPath, sUserID, sGroupID)
	return service.ContextWithDefaultRegistry(ctx)
}

func standaloneContext() (context.Context, error) { return nil, errFerryContextFactoryRequired }
func requireContextFactory() error                { return errFerryContextFactoryRequired }
