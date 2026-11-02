//go:build ferry_mobile

package libbox

import (
	"context"
	"github.com/sagernet/sing-box/daemon"
	"github.com/sagernet/sing-box/service/oomkiller"
	"github.com/sagernet/sing-box/service/powerreport"
)

// OOM and power profiling are unavailable in Ferry builds: the upstream
// profiler links private runtime/pprof symbols. Ordinary logging and crash
// redirection remain enabled independently in log_ferry.go and setup.go.
func (s *CommandServer) startReportRecorders(context.Context) {
	s.managedService = daemon.NewManagedService(daemon.ManagedServiceOptions{
		Handler: (*platformHandler)(s), Debug: sDebug,
	})
}

// Retain the Go-facing signatures without installing profiling callbacks.
func OOMRecorderOptions(*daemon.StartedService) oomkiller.RecorderOptions {
	return oomkiller.RecorderOptions{}
}
func PowerReportOptions(*daemon.StartedService) powerreport.Options {
	return powerreport.Options{}
}
func PromoteOOMDraft()         {}
func PromoteOOMDraftAt(string) {}
func DiscardPowerReportDraft() {}
