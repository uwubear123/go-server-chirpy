package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/uwubear123/go-server-chirpy/internal/auth"
	"github.com/uwubear123/go-server-chirpy/internal/database"
)

type parameters struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginParameters struct {
	parameters
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	params := parameters{}

	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Internal error", err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, 500, "Error hashing password", err)
		return
	}

	dbUser, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		respondWithError(w, 500, "Internal error", err)
		return
	}

	user := User{
		ID:          dbUser.ID.String(),
		CreatedAt:   dbUser.CreatedAt.String(),
		UpdatedAt:   dbUser.UpdatedAt.String(),
		Email:       dbUser.Email,
		IsChirpyRed: dbUser.IsChirpyRed,
	}

	respondWithJSON(w, 201, user)
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	params := loginParameters{}

	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Internal error", err)
		return
	}

	dbUser, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, 401, "Invalid credentials", nil)
		return
	}

	match, err := auth.CheckPassword(params.Password, dbUser.HashedPassword)
	if err != nil || !match {
		respondWithError(w, 401, "Invalid credentials", nil)
		return
	}

	token, err := auth.MakeJWT(dbUser.ID, cfg.secretKey, time.Hour)
	if err != nil {
		respondWithError(w, 500, "Error generating token", err)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, 500, "Error generating refresh token", err)
		return
	}

	_, err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:  refreshToken,
		UserID: dbUser.ID,
	})
	if err != nil {
		respondWithError(w, 500, "Error saving refresh token", err)
		return
	}

	user := User{
		ID:           dbUser.ID.String(),
		CreatedAt:    dbUser.CreatedAt.String(),
		UpdatedAt:    dbUser.UpdatedAt.String(),
		Email:        dbUser.Email,
		IsChirpyRed:  dbUser.IsChirpyRed,
		Token:        token,
		RefreshToken: refreshToken,
	}

	respondWithJSON(w, 200, user)
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token string `json:"token"`
	}

	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find token", nil)
		return
	}

	user, err := cfg.db.GetUserFromRefreshToken(r.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusUnauthorized, "Couldn't get user for refresh token", nil)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Internal error", err)
		return
	}

	accessToken, err := auth.MakeJWT(
		user.ID,
		cfg.secretKey,
		time.Hour,
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error generating token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		Token: accessToken,
	})
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find token", nil)
		return
	}

	err = cfg.db.DeleteRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't revoke session", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Missing or invalid token", nil)
		return
	}

	userID, err := auth.ValidateJWT(tokenString, cfg.secretKey)
	if err != nil {
		respondWithError(w, 401, "Missing or invalid token", nil)
		return
	}

	type updateUserParams struct {
		Email    string `json:"email,omitempty"`
		Password string `json:"password,omitempty"`
	}

	decoder := json.NewDecoder(r.Body)
	params := updateUserParams{}

	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Internal error", err)
		return
	}

	if params.Email == "" && params.Password == "" {
		respondWithError(w, 400, "No fields to update", nil)
		return
	}

	if params.Password != "" {
		hashedPassword, err := auth.HashPassword(params.Password)
		if err != nil {
			respondWithError(w, 500, "Error hashing password", err)
			return
		}
		params.Password = hashedPassword
	}

	dbUser, err := cfg.db.GetUserByID(r.Context(), userID)
	if err != nil {
		respondWithError(w, 500, "Internal error", err)
		return
	}

	newEmail := dbUser.Email
	if params.Email != "" {
		newEmail = params.Email
	}

	newHashedPassword := dbUser.HashedPassword
	if params.Password != "" {
		newHashedPassword = params.Password
	}

	updatedUser, err := cfg.db.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:             dbUser.ID,
		Email:          newEmail,
		HashedPassword: newHashedPassword,
	})
	if err != nil {
		respondWithError(w, 500, "Internal error", err)
		return
	}

	respondWithJSON(w, 200, User{
		ID:          updatedUser.ID.String(),
		CreatedAt:   updatedUser.CreatedAt.String(),
		UpdatedAt:   updatedUser.UpdatedAt.String(),
		Email:       updatedUser.Email,
		IsChirpyRed: updatedUser.IsChirpyRed,
	})
}
