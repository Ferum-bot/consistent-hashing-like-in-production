package api

import (
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/request"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/response"
)

type LocateAPI struct {
	routing RoutingService
}

func NewLocateAPI(routing RoutingService) *LocateAPI {
	return &LocateAPI{routing: routing}
}

func (a *LocateAPI) Get(w http.ResponseWriter, r *http.Request) {
	userID, err := request.PathInt64(r, "userID")
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	route, err := a.routing.Locate(r.Context(), userID)
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	response.JSON(w, http.StatusOK, toRouteDTO(route))
}
