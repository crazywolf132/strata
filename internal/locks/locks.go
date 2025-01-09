package locks

import (
	"strata/internal/logs"
	"sync"
	"time"
)

// We maintain a single lock to protect destructive operations in the repo.
// You could also have a per-repo lock if Strata can manage multiple repos concurrently.

var repoLock sync.Mutex

func LockRepo() {
	logs.Debug("Acquiring repo lock...")
	start := time.Now()
	repoLock.Lock()
	logs.Debug("Repo lock acquired (waited %v).", time.Since(start))
}

func UnlockRepo() {
	repoLock.Unlock()
	logs.Debug("Repo lock released.")
}
