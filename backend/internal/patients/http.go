package patients

import (
	"errors"
	"net/http"

	"github.com/eakillidev/Care-Flow/backend/internal/apiutil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) Create(w http.ResponseWriter, request *http.Request) {
	var input Input
	if err := apiutil.DecodeJSON(w, request, &input); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}
	patient, err := handler.service.Create(request.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusCreated, patient)
}

func (handler *Handler) List(w http.ResponseWriter, request *http.Request) {
	patients, err := handler.service.List(request.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, patients)
}

func (handler *Handler) Get(w http.ResponseWriter, request *http.Request) {
	id, err := uuid.Parse(chi.URLParam(request, "id"))
	if err != nil {
		apiutil.WriteError(w, http.StatusNotFound, "patient not found")
		return
	}
	patient, err := handler.service.Get(request.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, patient)
}

func (handler *Handler) Update(w http.ResponseWriter, request *http.Request) {
	id, err := uuid.Parse(chi.URLParam(request, "id"))
	if err != nil {
		apiutil.WriteError(w, http.StatusNotFound, "patient not found")
		return
	}
	var input Input
	if err := apiutil.DecodeJSON(w, request, &input); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}
	patient, err := handler.service.Update(request.Context(), id, input)
	if err != nil {
		handleError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, patient)
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidPatient):
		apiutil.WriteError(w, http.StatusBadRequest, "invalid patient")
	case errors.Is(err, ErrPatientNotFound):
		apiutil.WriteError(w, http.StatusNotFound, "patient not found")
	default:
		apiutil.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
