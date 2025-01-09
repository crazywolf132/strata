package net

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strata/internal/logs"
	"strata/internal/model"
	"sync"
	"time"
)

// In-memory store for ephemeral stacks (similar to local collab, but over HTTP).
// We also store a "token" => stack mapping, for more security.

var (
	storeMutex sync.RWMutex
	storeData  = map[string]model.StackTree{}
)

// StartServer starts an HTTP server with a few endpoints:
//   POST /share?token=XYZ => JSON body with entire stack
//   GET /share?token=XYZ => fetch stack
//   POST /rename => sync rename across ephemeral store

func StartServer(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/share", handleShare)
	mux.HandleFunc("/rename", handleRename)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	logs.Info("Enterprise server listening on :%d", port)
	return srv.ListenAndServe()
}

// If user hits POST /share?token=abc with JSON: { "stack": {...} }
// we store that stack in memory. GET /share?token=abc returns it.
func handleShare(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPost:
		var payload struct {
			Stack model.StackTree `json:"stack"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		storeMutex.Lock()
		storeData[token] = payload.Stack
		storeMutex.Unlock()
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	case http.MethodGet:
		storeMutex.RLock()
		st, ok := storeData[token]
		storeMutex.RUnlock()
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(st)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleRename for cross-ephemeral rename sync.
// e.g. POST /rename?token=abc => { "oldName": "...", "newName": "..." }
func handleRename(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token required", http.StatusBadRequest)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		OldName string `json:"oldName"`
		NewName string `json:"newName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	storeMutex.Lock()
	st, ok := storeData[token]
	if !ok {
		storeMutex.Unlock()
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if node, exists := st[body.OldName]; exists {
		node.BranchName = body.NewName
		st[body.NewName] = node
		delete(st, body.OldName)
		// update children references
		for _, n := range st {
			for i, c := range n.Children {
				if c == body.OldName {
					n.Children[i] = body.NewName
				}
			}
		}
		// update timestamps
		node.UpdatedAt = time.Now()
	}
	storeData[token] = st
	storeMutex.Unlock()
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

// utility function to generate random tokens if needed
func GenerateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
