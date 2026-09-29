package middlewares

import (
	"net/http"
	"strconv"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/headers"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/request"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/response"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/models"
)

func WithTopologyFence(fence TopologyFence) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			version, ok, err := request.HeaderInt64(r, headers.TopologyVersion)
			if err != nil {
				response.Error(w, http.StatusBadRequest, err)
				return
			}

			if !ok && fence.Strict() {
				w.Header().Set(headers.TopologyVersion, strconv.FormatInt(fence.Current(), 10))
				response.Error(w, http.StatusConflict, models.NewErrTopologyVersionRequired())

				return
			}

			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			current, err := fence.Observe(version)

			w.Header().Set(headers.TopologyVersion, strconv.FormatInt(current, 10))

			if err != nil {
				response.Error(w, http.StatusConflict, err)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
