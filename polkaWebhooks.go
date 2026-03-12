package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/uwubear123/go-server-chirpy/internal/auth"
)

type PolkaWebhookPayload struct {
	Event string `json:"event"`
	Data  struct {
		UserID string `json:"user_id"`
	} `json:"data"`
}

func (cfg *apiConfig) handlerPolkaWebhooks(w http.ResponseWriter, r *http.Request) {

	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, 401, "Missing or invalid API key", nil)
		return
	}

	if apiKey != cfg.polkaKey {
		respondWithError(w, 401, "Invalid API key", nil)
		return
	}

	decoder := json.NewDecoder(r.Body)
	payload := PolkaWebhookPayload{}
	err = decoder.Decode(&payload)
	if err != nil {
		respondWithError(w, 400, "Invalid JSON payload", err)
		return
	}

	if payload.Event != "user.upgraded" && payload.Event != "user_upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	id, err := uuid.Parse(payload.Data.UserID)
	if err != nil {
		respondWithError(w, 400, "Invalid user_id", nil)
		return
	}

	_, err = cfg.db.UpgradeUserToChirpyRed(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 404, "User not found", nil)
			return
		}
		respondWithError(w, 500, "Failed to upgrade user", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
