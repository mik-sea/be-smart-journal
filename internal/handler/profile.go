package handler

import (
	"encoding/json"
	"net/http"

	"smart-journal/internal/middleware"
	"smart-journal/internal/service"
)

type ProfileHandler struct {
	service *service.ProfileService
}

func NewProfileHandler(service *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{service: service}
}

func (h *ProfileHandler) Register(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("/api/profile", auth(http.HandlerFunc(h.profile)))
}

func (h *ProfileHandler) profile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	switch r.Method {
	case http.MethodGet:
		profile, err := h.service.Get(r.Context(), userID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, profile)
	case http.MethodPost:
		var request profileRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}

		profile, err := h.service.Ensure(r.Context(), service.CreateProfileInput{
			UserID:    userID,
			Email:     request.Email,
			Name:      request.Name,
			AvatarURL: request.AvatarURL,
			Bio:       request.Bio,
			Timezone:  request.Timezone,
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, profile)
	case http.MethodPatch, http.MethodPut:
		var request updateProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}

		profile, err := h.service.Update(r.Context(), service.UpdateProfileInput{
			UserID:    userID,
			Email:     request.Email,
			Name:      request.Name,
			AvatarURL: request.AvatarURL,
			Bio:       request.Bio,
			Timezone:  request.Timezone,
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, profile)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

type profileRequest struct {
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio"`
	Timezone  string `json:"timezone"`
}

type updateProfileRequest struct {
	Email     *string `json:"email"`
	Name      *string `json:"name"`
	AvatarURL *string `json:"avatar_url"`
	Bio       *string `json:"bio"`
	Timezone  *string `json:"timezone"`
}
