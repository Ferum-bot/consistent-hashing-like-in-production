package api

import (
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/request"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/response"
)

type FenceAPI struct {
	fence TopologyFence
}

func NewFenceAPI(fence TopologyFence) *FenceAPI {
	return &FenceAPI{fence: fence}
}

func (a *FenceAPI) Announce(w http.ResponseWriter, r *http.Request) {
	var req fenceRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	a.fence.SetStrict(req.Strict)

	response.JSON(w, http.StatusOK, fenceResponse{TopologyVersion: a.fence.Current(), Strict: a.fence.Strict()})
}
