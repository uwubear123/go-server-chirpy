package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/uwubear123/go-server-chirpy/internal/auth"
	"github.com/uwubear123/go-server-chirpy/internal/database"
)

type Chirp struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Body      string `json:"body"`
	UserID    string `json:"user_id"`
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Internal error", err)
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long", nil)
		return
	}

	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Missing or invalid token", nil)
		return
	}

	userID, err := auth.ValidateJWT(tokenString, cfg.secretKey)
	if err != nil {
		respondWithError(w, 401, "Invalid token", nil)
		return
	}

	cleanedBody := profanityFilter(params.Body)

	dbChirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   cleanedBody,
		UserID: userID,
	})
	if err != nil {
		respondWithError(w, 500, "Internal error", err)
		return
	}

	type chirpResponse struct {
		ID        string `json:"id"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Body      string `json:"body"`
		UserID    string `json:"user_id"`
	}

	respondWithJSON(w, 201, chirpResponse{
		ID:        dbChirp.ID.String(),
		CreatedAt: dbChirp.CreatedAt.String(),
		UpdatedAt: dbChirp.UpdatedAt.String(),
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID.String(),
	})
}

func (cfg *apiConfig) handlerListChirps(w http.ResponseWriter, r *http.Request) {
	authorIDStr := r.URL.Query().Get("author_id")
	authorID := uuid.Nil
	if authorIDStr != "" {
		parsedAuthorID, err := uuid.Parse(authorIDStr)
		if err != nil {
			respondWithError(w, 400, "Invalid author_id query parameter", err)
			return
		}
		authorID = parsedAuthorID
	}

	sortOrder := r.URL.Query().Get("sort")
	if sortOrder != "" && sortOrder != "asc" && sortOrder != "desc" {
		respondWithError(w, 400, "Invalid sort query parameter", nil)
		return
	}

	var (
		dbChirps []database.Chirp
		err      error
	)

	if authorID != uuid.Nil {
		dbChirps, err = cfg.db.ListChirpsByAuthor(r.Context(), authorID)
		if err != nil {
			respondWithError(w, 500, "Internal error", err)
			return
		}
	} else {
		dbChirps, err = cfg.db.ListChirps(r.Context())
		if err != nil {
			respondWithError(w, 500, "Internal error", err)
			return
		}
	}

	if sortOrder == "desc" {
		for i, j := 0, len(dbChirps)-1; i < j; i, j = i+1, j-1 {
			dbChirps[i], dbChirps[j] = dbChirps[j], dbChirps[i]
		}
	}

	chirps := make([]Chirp, 0, len(dbChirps))
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, Chirp{
			ID:        dbChirp.ID.String(),
			CreatedAt: dbChirp.CreatedAt.String(),
			UpdatedAt: dbChirp.UpdatedAt.String(),
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID.String(),
		})
	}

	respondWithJSON(w, 200, chirps)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("chirpID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, 400, "Invalid chirpID", err)
		return
	}
	dbChirp, err := cfg.db.GetChirp(r.Context(), id)
	if err != nil {
		respondWithError(w, 404, "Chirp not found", err)
		return
	}

	respondWithJSON(w, 200, Chirp{
		ID:        dbChirp.ID.String(),
		CreatedAt: dbChirp.CreatedAt.String(),
		UpdatedAt: dbChirp.UpdatedAt.String(),
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID.String(),
	})
}

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Missing or invalid token", nil)
		return
	}

	idStr := r.PathValue("chirpID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, 400, "Invalid chirpID", nil)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secretKey)
	if err != nil {
		respondWithError(w, 401, "Invalid token", nil)
		return
	}

	dbChirp, err := cfg.db.GetChirp(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 404, "Chirp not found", nil)
			return
		}
		respondWithError(w, 500, "Internal error", err)
		return
	}

	if dbChirp.UserID != userID {
		respondWithError(w, 403, "Forbidden", nil)
		return
	}

	err = cfg.db.DeleteChirp(r.Context(), id)
	if err != nil {
		respondWithError(w, 500, "Internal error", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
