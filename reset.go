package main

import (
	"context"
	"net/http"
)

func (cfg *apiConfig) handlerResetMetrics(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden"))
		return
	}

	err := cfg.db.DeleteAllUsers(context.Background())
	if err != nil {
		respondWithError(w, 500, "Failed to delete users", err)
		return
	}

	cfg.fileserverHits.Store(0)
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}
