//go:build ferry_mobile && (darwin || linux || windows)

package libbox

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"
)

type crashReportMetadata struct {
	reportMetadata
	CrashedAt       string `json:"crashedAt,omitempty"`
	SignalName      string `json:"signalName,omitempty"`
	SignalCode      string `json:"signalCode,omitempty"`
	ExceptionName   string `json:"exceptionName,omitempty"`
	ExceptionReason string `json:"exceptionReason,omitempty"`
}

func archiveCrashReport(path string, crashReportsDir string) {
	content, err := os.ReadFile(path)
	if err != nil || len(content) == 0 {
		return
	}

	info, _ := os.Stat(path)
	crashTime := time.Now().UTC()
	if info != nil {
		crashTime = info.ModTime().UTC()
	}

	initReportDir(crashReportsDir)
	destPath, err := nextAvailableReportPath(crashReportsDir, crashTime)
	if err != nil {
		return
	}
	initReportDir(destPath)

	writeReportFile(destPath, "go.log", content)
	metadata := crashReportMetadata{
		reportMetadata: baseReportMetadata(),
		CrashedAt:      crashTime.Format(time.RFC3339),
	}
	platformSnapshot := readPlatformSnapshot()
	if platformSnapshot != nil {
		metadata.Platform = platformSnapshot
	}
	writeReportMetadata(destPath, metadata)
	os.Remove(path)
	copyConfigSnapshot(destPath)
}

func configSnapshotPath() string {
	return filepath.Join(sBasePath, "configuration.json")
}

func saveConfigSnapshot(configContent string) {
	snapshotPath := configSnapshotPath()
	os.WriteFile(snapshotPath, []byte(configContent), 0o666)
	chownReport(snapshotPath)
}

func redirectStderr(path string) error {
	crashReportsDir := filepath.Join(sWorkingPath, "crash_reports")
	archiveCrashReport(path, crashReportsDir)
	archiveCrashReport(path+".old", crashReportsDir)

	outputFile, err := os.Create(path)
	if err != nil {
		return err
	}
	if runtime.GOOS != "android" && runtime.GOOS != "windows" {
		err = outputFile.Chown(sUserID, sGroupID)
		if err != nil {
			outputFile.Close()
			os.Remove(outputFile.Name())
			return err
		}
	}

	err = debug.SetCrashOutput(outputFile, debug.CrashOptions{})
	if err != nil {
		outputFile.Close()
		os.Remove(outputFile.Name())
		return err
	}
	_ = outputFile.Close()
	return nil
}

// CreateZipArchive intentionally rejects both archive modes. Never silently
// downgrade an encrypted diagnostic export to plaintext in a Ferry build.
func CreateZipArchive(sourcePath string, destinationPath string, encrypt bool) error {
	return errFerryUnsupported
}
