//go:build !ferry_mobile

package libbox

import (
	"context"
	"github.com/sagernet/sing-box/daemon"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/service/oomkiller"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/service"
)

func (s *CommandServer) startReportRecorders(ctx context.Context) {
	recorder := oomkiller.NewRecorder(OOMRecorderOptions(s.StartedService))
	service.MustRegister[*oomkiller.Recorder](ctx, recorder)
	recorder.Start()
	s.oomRecorder = recorder
	s.managedService = daemon.NewManagedService(daemon.ManagedServiceOptions{
		Handler: (*platformHandler)(s), Debug: sDebug, OOMRecorder: recorder,
	})
	if sPowerReportEnabled {
		if err := s.powerManager.Start(PowerReportOptions(s.StartedService)); err != nil {
			log.StdLogger().Error(E.Cause(err, "start power report recorder"))
		}
	}
}
