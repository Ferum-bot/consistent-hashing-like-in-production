package api

import (
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/request"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/response"
)

type ConfigAPI struct {
	generator GeneratorService
}

func NewConfigAPI(generator GeneratorService) *ConfigAPI {
	return &ConfigAPI{generator: generator}
}

func (a *ConfigAPI) Get(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, toProfileDTO(a.generator.Profile()))
}

func (a *ConfigAPI) Patch(w http.ResponseWriter, r *http.Request) {
	var patch profilePatchDTO
	if err := request.DecodeJSON(r, &patch); err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	profile, err := a.generator.UpdateProfile(patch.applyTo)
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	response.JSON(w, http.StatusOK, toProfileDTO(profile))
}
