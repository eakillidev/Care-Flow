package auth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/eakillidev/Care-Flow/backend/internal/users"
)

const maxLoginBodyBytes = 1 << 20

type Handler struct {
	service *Service
	users   users.Repository
}

func NewHandler(service *Service, userRepository users.Repository) *Handler {
	return &Handler{service: service, users: userRepository}
}

func (handler *Handler) Login(w http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(w, request.Body, maxLoginBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	var input LoginRequest
	if err := decoder.Decode(&input); err != nil || strings.TrimSpace(input.Email) == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	response, err := handler.service.Login(request.Context(), input.Email, input.Password)
	if err != nil {
		if err == ErrInvalidCredentials {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (handler *Handler) Me(w http.ResponseWriter, request *http.Request) {
	identity, ok := IdentityFromContext(request.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	user, err := handler.users.GetByID(request.Context(), identity.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, NewUserResponse(user))
}

func CoordinatorPing(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "coordinator access granted"})
}

func CaregiverPing(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "caregiver access granted"})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
