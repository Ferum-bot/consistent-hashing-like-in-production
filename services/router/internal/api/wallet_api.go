package api

import (
	"errors"
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/headers"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/request"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/response"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

const retryAfterSeconds = "1"

type WalletAPI struct {
	routing RoutingService
	proxy   Proxy
}

func NewWalletAPI(routing RoutingService, proxy Proxy) *WalletAPI {
	return &WalletAPI{routing: routing, proxy: proxy}
}

func (a *WalletAPI) Read(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set(headers.RoutedTo, route.Node.ID)

	a.proxy.Forward(w, r, route.Node, route.FenceVersion)
}

func (a *WalletAPI) Write(w http.ResponseWriter, r *http.Request) {
	userID, err := request.PathInt64(r, "userID")
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	route, release, err := a.routing.LocateWrite(r.Context(), userID)
	if errors.Is(err, models.ErrMigrationTimeout) {
		w.Header().Set(headers.RetryAfter, retryAfterSeconds)
	}

	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}
	defer release()

	w.Header().Set(headers.RoutedTo, route.Node.ID)

	a.proxy.Forward(w, r, route.Node, route.FenceVersion)
}
