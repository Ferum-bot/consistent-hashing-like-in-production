package api

import (
	"net/http"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/request"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/response"
)

type TopologyAPI struct {
	topology TopologyService
}

func NewTopologyAPI(topology TopologyService) *TopologyAPI {
	return &TopologyAPI{topology: topology}
}

func (a *TopologyAPI) Get(w http.ResponseWriter, r *http.Request) {
	a.writeCurrent(w, r)
}

func (a *TopologyAPI) Apply(w http.ResponseWriter, r *http.Request) {
	topology, err := decodeTopology(r)
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	deferred, err := a.topology.Apply(r.Context(), topology)
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	a.writeResult(w, r, deferred)
}

func (a *TopologyAPI) Prepare(w http.ResponseWriter, r *http.Request) {
	topology, err := decodeTopology(r)
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	deferred, err := a.topology.Prepare(r.Context(), topology)
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	a.writeResult(w, r, deferred)
}

func (a *TopologyAPI) Commit(w http.ResponseWriter, r *http.Request) {
	var req commitRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	deferred, err := a.topology.Commit(r.Context(), req.Version)
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	a.writeResult(w, r, deferred)
}

func (a *TopologyAPI) Abort(w http.ResponseWriter, r *http.Request) {
	deferred, err := a.topology.Abort(r.Context())
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	a.writeResult(w, r, deferred)
}

func (a *TopologyAPI) writeResult(w http.ResponseWriter, r *http.Request, deferred time.Duration) {
	if deferred > 0 {
		response.JSON(w, http.StatusAccepted, deferredResponse{DeferredFor: deferred.String()})
		return
	}

	a.writeCurrent(w, r)
}

func (a *TopologyAPI) writeCurrent(w http.ResponseWriter, r *http.Request) {
	state, err := a.topology.State(r.Context())
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	response.JSON(w, http.StatusOK, toStateDTO(state))
}
