package caregivers

import (
	"net/http"

	"github.com/eakillidev/Care-Flow/backend/internal/apiutil"
	"github.com/eakillidev/Care-Flow/backend/internal/auth"
	"github.com/eakillidev/Care-Flow/backend/internal/users"
)

type Handler struct {
	users users.Repository
}

func NewHandler(userRepository users.Repository) *Handler {
	return &Handler{users: userRepository}
}

func (handler *Handler) List(w http.ResponseWriter, request *http.Request) {
	caregiverUsers, err := handler.users.ListByRole(request.Context(), users.RoleCaregiver)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response := make([]auth.UserResponse, 0, len(caregiverUsers))
	for index := range caregiverUsers {
		response = append(response, auth.NewUserResponse(&caregiverUsers[index]))
	}
	apiutil.WriteJSON(w, http.StatusOK, response)
}
