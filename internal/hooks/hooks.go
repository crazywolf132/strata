package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strata/internal/config"
	"strata/internal/logs"
	"strings"
	"time"
)

func ListHooks() []string {
	raw := config.GetConfigValue("hooks")
	if raw == "" {
		return []string{}
	}
	return strings.Split(raw, ";")
}

func AddHook(event, scriptPath string) error {
	if event == "" || scriptPath == "" {
		return fmt.Errorf("invalid hook parameters")
	}

	hs := ListHooks()
	newHook := fmt.Sprintf("%s|%s", event, scriptPath)
	hs = append(hs, newHook)
	joined := strings.Join(hs, ";")
	return config.SetConfigValue("hooks", joined, false)
}

func RunHooks(event, arg string) {
	hs := ListHooks()
	for _, h := range hs {
		parts := strings.SplitN(h, "|", 2)
		if len(parts) != 2 {
			logs.Warn("Invalid hook format: '%s'", h)
			continue
		}
		if parts[0] == event {
			runHookScript(parts[1], event, arg)
		}
	}
}

func runHookScript(script, event, arg string) {
	abs, err := filepath.Abs(script)
	if err != nil {
		logs.Warn("Failed to get absolute path for hook script '%s': %v", script, err)
		return
	}

	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		logs.Warn("Hook script not found or is a directory: '%s'", abs)
		return
	}
	logs.Debug("Running hook script '%s' for event '%s'", abs, event)

	cmd := exec.Command(abs, event, arg)

	// Optional: set a timeout
	done := make(chan error, 1)
	go func() {
		out, e := cmd.CombinedOutput()
		if e != nil {
			logs.Warn("Hook script '%s' failed: %v\nOutput: %s", abs, e, string(out))
		} else {
			logs.Info("Hook script '%s' executed successfully.\nOutput: %s", abs, string(out))
		}
		done <- e
	}()

	select {
	case <-time.After(30 * time.Second):
		logs.Warn("Hook script '%s' timed out after 30s.", abs)
		// Attempt to kill the process
		_ = cmd.Process.Kill()
	case e := <-done:
		if e != nil {
			logs.Warn("Hook script '%s' ended with error: %v", abs, e)
		}
	}
}
