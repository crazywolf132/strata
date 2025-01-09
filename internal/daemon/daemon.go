package daemon

import (
	"fmt"
	"strata/internal/config"
	"strata/internal/logs"
	"strata/internal/service"
	"sync"
	"time"
)

var (
	monitoredRepos = make(map[string]bool)
	mu             sync.Mutex
	running        bool

	// pollInterval for server sync
	pollInterval = 15 * time.Second
)

func IsDaemonRunning() bool {
	return running
}

// RegisterRepo is called during `strata init` if the daemon is running.
func RegisterRepo() error {
	repoName := config.GetConfigValue("repo_name")
	if repoName == "" {
		repoName = "unknown-repo"
	}

	mu.Lock()
	monitoredRepos[repoName] = true
	mu.Unlock()

	logs.Info("[Daemon] Registered repo: %s", repoName)
	return nil
}

// Run starts the daemon in this process (blocking).
func Run() error {
	if running {
		return fmt.Errorf("daemon is already running")
	}
	running = true
	defer func() { running = false }()

	logs.Info("[Daemon] Strata daemon started (monitoring for shared stacks).")

	for {
		mu.Lock()
		// copy the monitored repos
		reposToCheck := make([]string, 0, len(monitoredRepos))
		for r := range monitoredRepos {
			reposToCheck = append(reposToCheck, r)
		}
		mu.Unlock()

		for _, r := range reposToCheck {
			logs.Debug("[Daemon] Checking for changes in repo: %s", r)
			// The actual "repoName" is basically for logging or advanced multi-repo scenario.
			// For now, we'll assume there's only one local repo unless we expand Strata to truly handle multiple repos in a single daemon instance.

			syncSharedStackIfNeeded()
		}

		time.Sleep(pollInterval)
	}
}

// syncSharedStackIfNeeded detects if the local stack is “shared” (has a share code or server token).
// If it is, we push local changes to the server & pull remote changes from the server.
func syncSharedStackIfNeeded() {
	collabSvc := service.GetCollabService()
	if !collabSvc.HasServerOrShare() {
		// means we do not have a share code or server token => no sync
		logs.Debug("[Daemon] No shared stack found; skipping sync.")
		return
	}

	logs.Info("[Daemon] Found shared stack. Syncing with server or ephemeral store...")

	// 1. Push local changes to server
	if err := collabSvc.PushLocalToServer(); err != nil {
		logs.Warn("[Daemon] Failed to push local changes to server: %v", err)
	}

	// 2. Pull remote changes from server
	if err := collabSvc.PullServerToLocal(); err != nil {
		logs.Warn("[Daemon] Failed to pull remote changes from server: %v", err)
	}

	logs.Info("[Daemon] Sync complete for shared stack.")
}
