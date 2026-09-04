//go:build windows

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func installUpdate(source, target string, pid int) error {
	updateDir := filepath.Dir(source)
	scriptPath := filepath.Join(updateDir, "apply-update.cmd")
	script := fmt.Sprintf(`@echo off
setlocal
set "SOURCE=%s"
set "TARGET=%s"
:wait
tasklist /FI "PID eq %d" /NH | findstr /R /C:" %d " >NUL
if not errorlevel 1 (
  timeout /t 1 /nobreak >NUL
  goto wait
)
copy /Y "%%SOURCE%%" "%%TARGET%%" >NUL
if errorlevel 1 exit /b 1
start "" "%%TARGET%%"
del "%%SOURCE%%" >NUL 2>&1
del "%%~f0" >NUL 2>&1
`, cmdValue(source), cmdValue(target), pid, pid)
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		return fmt.Errorf("create update installer: %w", err)
	}
	if err := exec.Command("cmd.exe", "/C", "call", scriptPath).Start(); err != nil {
		return fmt.Errorf("start update installer: %w", err)
	}
	return nil
}

func cmdValue(value string) string {
	return strings.ReplaceAll(value, `"`, `""`)
}
