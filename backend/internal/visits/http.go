package visits

import (
	"errors"
	"net/http"

	"github.com/eakillidev/Care-Flow/backend/internal/apiutil"
	"github.com/eakillidev/Care-Flow/backend/internal/auth"
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
	var input CreateInput
	if err := apiutil.DecodeJSON(w, request, &input); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}
	detail, err := handler.service.Create(request.Context(), input)
	if err != nil {
		handleHTTPError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusCreated, detail)
}

func (handler *Handler) List(w http.ResponseWriter, request *http.Request) {
	details, err := handler.service.List(request.Context())
	if err != nil {
		handleHTTPError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, details)
}

func (handler *Handler) Get(w http.ResponseWriter, request *http.Request) {
	id, ok := visitID(w, request)
	if !ok {
		return
	}
	detail, err := handler.service.Get(request.Context(), id)
	if err != nil {
		handleHTTPError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, detail)
}

func (handler *Handler) UpdateSchedule(w http.ResponseWriter, request *http.Request) {
	id, ok := visitID(w, request)
	if !ok {
		return
	}
	var input UpdateScheduleInput
	if err := apiutil.DecodeJSON(w, request, &input); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}
	detail, err := handler.service.UpdateSchedule(request.Context(), id, input)
	if err != nil {
		handleHTTPError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, detail)
}

func (handler *Handler) Cancel(w http.ResponseWriter, request *http.Request) {
	id, ok := visitID(w, request)
	if !ok {
		return
	}
	detail, err := handler.service.Cancel(request.Context(), id)
	if err != nil {
		handleHTTPError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, detail)
}

func (handler *Handler) ListForCaregiver(w http.ResponseWriter, request *http.Request) {
	identity, ok := auth.IdentityFromContext(request.Context())
	if !ok {
		apiutil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	details, err := handler.service.ListForCaregiver(request.Context(), identity.UserID)
	if err != nil {
		handleHTTPError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, details)
}

func (handler *Handler) GetForCaregiver(w http.ResponseWriter, request *http.Request) {
	identity, ok := auth.IdentityFromContext(request.Context())
	if !ok {
		apiutil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, valid := visitID(w, request)
	if !valid {
		return
	}
	detail, err := handler.service.GetForCaregiver(request.Context(), id, identity.UserID)
	if err != nil {
		handleHTTPError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, detail)
}

func visitID(w http.ResponseWriter, request *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(request, "id"))
	if err != nil {
		apiutil.WriteError(w, http.StatusNotFound, "visit not found")
		return uuid.Nil, false
	}
	return id, true
}

func handleHTTPError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidSchedule):
		apiutil.WriteError(w, http.StatusBadRequest, ErrInvalidSchedule.Error())
	case errors.Is(err, ErrPatientNotFound):
		apiutil.WriteError(w, http.StatusNotFound, ErrPatientNotFound.Error())
	case errors.Is(err, ErrCaregiverNotFound):
		apiutil.WriteError(w, http.StatusNotFound, ErrCaregiverNotFound.Error())
	case errors.Is(err, ErrAssignedUserNotCaregiver):
		apiutil.WriteError(w, http.StatusBadRequest, ErrAssignedUserNotCaregiver.Error())
	case errors.Is(err, ErrOverlappingVisit):
		apiutil.WriteError(w, http.StatusConflict, ErrOverlappingVisit.Error())
	case errors.Is(err, ErrVisitNotFound):
		apiutil.WriteError(w, http.StatusNotFound, ErrVisitNotFound.Error())
	case errors.Is(err, ErrVisitNotSchedulable):
		apiutil.WriteError(w, http.StatusConflict, ErrVisitNotSchedulable.Error())
	case errors.Is(err, ErrVisitNotCancellable):
		apiutil.WriteError(w, http.StatusConflict, ErrVisitNotCancellable.Error())
	default:
		apiutil.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
