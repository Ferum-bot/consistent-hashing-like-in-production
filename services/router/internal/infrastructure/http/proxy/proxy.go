package proxy

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/headers"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/recorder"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/response"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

type targetKey struct{}

type target struct {
	node    sharding.Node
	url     *url.URL
	version int64
}

type Proxy struct {
	reverse  *httputil.ReverseProxy
	observer UpstreamObserver
}

func New(transport http.RoundTripper, observer UpstreamObserver) *Proxy {
	reverse := &httputil.ReverseProxy{
		Transport: transport,
		Rewrite: func(pr *httputil.ProxyRequest) {
			to := targetOf(pr.In)

			pr.SetURL(to.url)
			pr.Out.Header.Del(headers.TopologyVersion)

			if to.version > 0 {
				pr.Out.Header.Set(headers.TopologyVersion, strconv.FormatInt(to.version, 10))
			}
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			response.Error(w, http.StatusBadGateway, models.NewErrNodeUnavailable(targetOf(r).node.ID, err))
		},
	}

	return &Proxy{reverse: reverse, observer: observer}
}

func (p *Proxy) Forward(w http.ResponseWriter, r *http.Request, node sharding.Node, version int64) {
	addr, err := url.Parse(node.Addr)
	if err != nil {
		response.Error(w, http.StatusBadGateway, models.NewErrNodeUnavailable(node.ID, err))
		return
	}

	start := time.Now()
	rec := recorder.New(w)
	ctx := context.WithValue(r.Context(), targetKey{}, target{node: node, url: addr, version: version})

	p.reverse.ServeHTTP(rec, r.WithContext(ctx))

	p.observer.ObserveUpstream(node.ID, rec.Status(), time.Since(start))
}

func targetOf(r *http.Request) target {
	return r.Context().Value(targetKey{}).(target)
}
