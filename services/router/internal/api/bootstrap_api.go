package api

import (
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/response"
)

type BootstrapAPI struct {
	bootstrap BootstrapService
}

func NewBootstrapAPI(bootstrap BootstrapService) *BootstrapAPI {
	return &BootstrapAPI{bootstrap: bootstrap}
}

func (a *BootstrapAPI) Run(w http.ResponseWriter, r *http.Request) {
	result, err := a.bootstrap.Run(r.Context())
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	response.JSON(w, http.StatusOK, toBootstrapResponse(result))
}
