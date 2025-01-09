package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strata/internal/config"
	"strata/internal/logs"
	"strata/internal/model"
	"strata/internal/store"
	"strata/internal/utils"
	"sync"
)

type CollabService struct {
	// Possibly store the serverURL or shareCode if we have them
	serverURL  string
	shareCode  string
	shareMux   sync.RWMutex
	localStore model.StackTree // ephemeral store if no serverURL
}

var (
	collabSvc    *CollabService
	ephemeralMap = make(map[string]model.StackTree)
)

// GetCollabService is a singleton accessor
func GetCollabService() *CollabService {
	if collabSvc == nil {
		collabSvc = &CollabService{}
		// If global config has something like "server_url" => set serverURL
		url := config.GetConfigValue("server_url")
		if url != "" {
			collabSvc.serverURL = url
		}
	}
	return collabSvc
}

// HasServerOrShare returns true if we have a serverURL or a share code
func (c *CollabService) HasServerOrShare() bool {
	// A stack is considered "shared" if we have either a serverURL set, or if we have a local ephemeral share code
	if c.serverURL != "" {
		return true
	}
	// Or if we have ephemeral share code already stored
	c.shareMux.RLock()
	has := (c.shareCode != "")
	c.shareMux.RUnlock()
	return has
}

// GenerateShareCode copies the local stack into ephemeral memory for others to pull.
func (c *CollabService) GenerateShareCode() (string, error) {
	localStack := GetStackService().GetStack()
	code := utils.RandomShareCode()

	c.shareMux.Lock()
	c.shareCode = code
	ephemeralMap[code] = cloneStack(localStack)
	c.shareMux.Unlock()

	logs.Info("[Collab] Generated share code '%s' with a copy of the local stack", code)
	return code, nil
}

// PullSharedStack merges the ephemeral stack from code into the local stack. (Peer-to-peer style)
func (c *CollabService) PullSharedStack(code string) error {
	c.shareMux.Lock()
	c.shareCode = code
	c.shareMux.Unlock()

	ephemeralMap[c.shareCode] = ephemeralMap[c.shareCode] // in case no changes

	st, ok := ephemeralMap[code]
	if !ok {
		return fmt.Errorf("no shared stack found for code '%s'", code)
	}

	logs.Info("[Collab] Pulling shared stack from code '%s'", code)
	localSvc := GetStackService()
	localSt := localSvc.GetStack()

	// Merge ephemeral stack => local
	for k, v := range st {
		localSt[k] = v
	}
	err := storeAndRefresh(localSvc, localSt)
	if err != nil {
		return err
	}

	logs.Info("[Collab] Shared stack pulled successfully from code '%s'", code)
	return nil
}

// PushLocalToServer pushes the local stack to the server (if serverURL is set).
// If we only have a share code, we do ephemeralMap sync instead.
func (c *CollabService) PushLocalToServer() error {
	localSvc := GetStackService()
	localStack := cloneStack(localSvc.GetStack())

	// If no serverURL => ephemeral sync
	if c.serverURL == "" {
		c.shareMux.Lock()
		if c.shareCode == "" {
			// no share code => no sync
			logs.Debug("[Collab] No share code => nothing to push.")
			c.shareMux.Unlock()
			return nil
		}
		ephemeralMap[c.shareCode] = localStack
		c.shareMux.Unlock()
		logs.Info("[Collab] Successfully pushed local stack to ephemeral store under code '%s'", c.shareCode)
		return nil
	}

	// We have serverURL => let's push via HTTP POST /share?token=<token>
	token := getServerToken()
	if token == "" {
		return fmt.Errorf("no server token found; can't push to server")
	}

	url := fmt.Sprintf("%s/share?token=%s", c.serverURL, token)
	payload := struct {
		Stack model.StackTree `json:"stack"`
	}{
		Stack: localStack,
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to push to server: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("server responded with status %d", resp.StatusCode)
	}
	logs.Info("[Collab] Successfully pushed local stack to server at %s", url)
	return nil
}

// PullServerToLocal fetches remote stack from the server or ephemeral store and merges it in.
func (c *CollabService) PullServerToLocal() error {
	localSvc := GetStackService()
	localSt := localSvc.GetStack()

	// If no server => ephemeral
	if c.serverURL == "" {
		c.shareMux.RLock()
		code := c.shareCode
		c.shareMux.RUnlock()

		if code == "" {
			logs.Debug("[Collab] No share code => nothing to pull.")
			return nil
		}
		remoteSt, ok := ephemeralMap[code]
		if !ok {
			return fmt.Errorf("share code '%s' not found in ephemeral store", code)
		}
		// Merge in
		for k, v := range remoteSt {
			localSt[k] = v
		}
		return storeAndRefresh(localSvc, localSt)
	}

	// If we do have a server URL
	token := getServerToken()
	if token == "" {
		return fmt.Errorf("no server token found; can't pull from server")
	}
	url := fmt.Sprintf("%s/share?token=%s", c.serverURL, token)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch from server: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("server responded with status %d", resp.StatusCode)
	}
	var remotePayload model.StackTree
	if e := json.NewDecoder(resp.Body).Decode(&remotePayload); e != nil {
		return fmt.Errorf("invalid JSON from server: %v", e)
	}
	// merge
	for k, v := range remotePayload {
		localSt[k] = v
	}
	if err := storeAndRefresh(localSvc, localSt); err != nil {
		return err
	}
	logs.Info("[Collab] Successfully pulled remote stack from server at %s", url)
	return nil
}

// getServerToken might read from config if the user has e.g. "server_token" or "token" set
func getServerToken() string {
	t := config.GetConfigValue("server_token")
	return t
}

func cloneStack(st model.StackTree) model.StackTree {
	newSt := make(model.StackTree)
	for k, v := range st {
		cpy := *v
		cKids := make([]string, len(v.Children))
		copy(cKids, v.Children)
		cpy.Children = cKids
		newSt[k] = &cpy
	}
	return newSt
}

func storeAndRefresh(svc *StackService, updated model.StackTree) error {
	svc.stack = updated
	// Save to disk
	if err := store.SaveStack(updated); err != nil {
		return err
	}
	// Reload in case the file changes externally
	return svc.ReloadStack()
}
