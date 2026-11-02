//go:build ferry_mobile

package libbox

import (
	"context"
	"github.com/sagernet/sing-box/daemon"
	"github.com/sagernet/sing-box/service/powerreport"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"testing"
)

func TestFerryReportsDisabled(t *testing.T) {
	s := &CommandServer{powerManager: powerreport.NewManager()}
	s.startReportRecorders(context.Background())
	if s.oomRecorder != nil || s.powerManager.Recorder() != nil {
		t.Fatal("profiling enabled")
	}
	managed := daemon.NewManagedService(daemon.ManagedServiceOptions{})
	_, err := managed.TriggerOOMReport(context.Background(), &emptypb.Empty{})
	if status.Code(err) != codes.Unavailable {
		t.Fatal(err)
	}
	if OOMRecorderOptions(nil).SnapshotCallback != nil || PowerReportOptions(nil).ProfileCallback != nil {
		t.Fatal("profiling callbacks installed")
	}
	PromoteOOMDraft()
	PromoteOOMDraftAt(t.TempDir())
	DiscardPowerReportDraft()
}
