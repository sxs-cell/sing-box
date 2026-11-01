//go:build ferry_mobile

package libbox

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFerryRequiresFactoryBeforeServerSideEffects(t *testing.T) {
	dir := testPaths(t)
	for _, platform := range []PlatformInterface{nil, &testPlatform{}} {
		s, err := NewCommandServer(nil, platform)
		if s != nil || !errors.Is(err, errFerryContextFactoryRequired) {
			t.Fatal("default factory fallback", s, err)
		}
	}
	for _, check := range []func() error{
		func() error { return CheckConfig("{}") },
		func() error { _, err := FormatConfig("{}"); return err },
		func() error { _, err := GenerateConfigSchema(); return err },
		func() error { _, err := HasTunInbound("{}"); return err },
	} {
		if err := check(); !errors.Is(err, errFerryContextFactoryRequired) {
			t.Fatal(err)
		}
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 0 {
		t.Fatal("missing factory caused filesystem side effects", err, files)
	}
}

func TestFerryExcludedEntrypointsFailWithoutCreatingFiles(t *testing.T) {
	dir := testPaths(t)
	if shell, err := OpenNativeShellSession("", "", nil, nil, "", 0, 0, 0, 0, nil); shell != nil || !errors.Is(err, errFerryUnsupported) {
		t.Fatal(shell, err)
	}
	if shell, err := OpenNativePipeSession("", "", nil, nil, 0, 0, nil); shell != nil || !errors.Is(err, errFerryUnsupported) {
		t.Fatal(shell, err)
	}
	for _, encrypt := range []bool{true, false} {
		output := filepath.Join(dir, "report.zip")
		err := CreateZipArchive(dir, output, encrypt)
		if !errors.Is(err, errFerryUnsupported) || !strings.Contains(err.Error(), "本构建不支持") {
			t.Fatal(err)
		}
		if _, err = os.Stat(output); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("unsupported archive created output", err)
		}
	}
	// Required non-age snapshot helpers still work; Setup/report references were
	// preserved rather than silently replaced with no-ops.
	saveConfigSnapshot("fixture")
	data, err := os.ReadFile(configSnapshotPath())
	if err != nil || string(data) != "fixture" {
		t.Fatal("snapshot helper lost", err)
	}
}
