package api

import (
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/response"
)

type StatsAPI struct {
	nodeID string
	stats  StatsService
	fence  TopologyFence
}

func NewStatsAPI(nodeID string, stats StatsService, fence TopologyFence) *StatsAPI {
	return &StatsAPI{nodeID: nodeID, stats: stats, fence: fence}
}

func (a *StatsAPI) Get(w http.ResponseWriter, r *http.Request) {
	stats, err := a.stats.Stats(r.Context())
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	response.JSON(w, http.StatusOK, statsResponse{
		Node:            a.nodeID,
		TopologyVersion: a.fence.Current(),
		Strict:          a.fence.Strict(),
		Users:           stats.Users,
		Wallets:         stats.Wallets,
	})
}
