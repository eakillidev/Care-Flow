package visits

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/eakillidev/Care-Flow/backend/internal/apiutil"
	"github.com/eakillidev/Care-Flow/backend/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

type locationInput struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
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
	filter, err := parseFilter(request, true)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid filters")
		return
	}
	details, err := handler.service.ListFiltered(request.Context(), filter)
	if err != nil {
		handleHTTPError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, details)
}

func (handler *Handler) EVVSummary(w http.ResponseWriter, request *http.Request) {
	filter, err := parseFilter(request, false)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid filters")
		return
	}
	summary, err := handler.service.Summary(request.Context(), filter)
	if err != nil {
		handleHTTPError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, summary)
}

func parseFilter(request *http.Request, all bool) (Filter, error) {
	query := request.URL.Query()
	var filter Filter
	if all && query.Get("status") != "" {
		value := Status(query.Get("status"))
		if value != StatusScheduled && value != StatusInProgress && value != StatusCompleted && value != StatusCancelled {
			return filter, errors.New("invalid status")
		}
		filter.Status = &value
	}
	if all && query.Get("evv_status") != "" {
		value := EVVStatus(query.Get("evv_status"))
		if value != EVVStatusPending && value != EVVStatusVerified && value != EVVStatusException {
			return filter, errors.New("invalid EVV status")
		}
		filter.EVVStatus = &value
	}
	if all && query.Get("caregiver_id") != "" {
		value, err := uuid.Parse(query.Get("caregiver_id"))
		if err != nil {
			return filter, err
		}
		filter.CaregiverID = &value
	}
	if all && query.Get("patient_id") != "" {
		value, err := uuid.Parse(query.Get("patient_id"))
		if err != nil {
			return filter, err
		}
		filter.PatientID = &value
	}
	for key, target := range map[string]**time.Time{"from": &filter.From, "to": &filter.To} {
		if query.Get(key) == "" {
			continue
		}
		value, err := time.Parse("2006-01-02", query.Get(key))
		if err != nil {
			return filter, err
		}
		if key == "to" {
			value = value.AddDate(0, 0, 1)
		}
		*target = &value
	}
	if filter.From != nil && filter.To != nil && !filter.From.Before(*filter.To) {
		return filter, errors.New("invalid date range")
	}
	return filter, nil
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

func (handler *Handler) CheckIn(w http.ResponseWriter, request *http.Request) {
	handler.handleEVV(w, request, handler.service.CheckIn)
}

func (handler *Handler) CheckOut(w http.ResponseWriter, request *http.Request) {
	handler.handleEVV(w, request, handler.service.CheckOut)
}

func (handler *Handler) handleEVV(
	w http.ResponseWriter,
	request *http.Request,
	action func(context.Context, uuid.UUID, uuid.UUID, float64, float64) (*EVVResponse, error),
) {
	identity, ok := auth.IdentityFromContext(request.Context())
	if !ok {
		apiutil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, valid := visitID(w, request)
	if !valid {
		return
	}
	var input locationInput
	if err := apiutil.DecodeJSON(w, request, &input); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}
	response, err := action(request.Context(), id, identity.UserID, input.Latitude, input.Longitude)
	if err != nil {
		handleHTTPError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, response)
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
	case errors.Is(err, ErrInvalidCoordinates):
		apiutil.WriteError(w, http.StatusBadRequest, ErrInvalidCoordinates.Error())
	case errors.Is(err, ErrVisitNotAvailableForCheckIn):
		apiutil.WriteError(w, http.StatusConflict, ErrVisitNotAvailableForCheckIn.Error())
	case errors.Is(err, ErrVisitNotInProgress):
		apiutil.WriteError(w, http.StatusConflict, ErrVisitNotInProgress.Error())
	default:
		apiutil.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
