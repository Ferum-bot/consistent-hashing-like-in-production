package client

import (
	"net/http"
	"time"
)

const (
	maxIdleConns        = 1024
	maxIdleConnsPerHost = 256
	idleConnTimeout     = 90 * time.Second
)

func NewTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	transport.MaxIdleConns = maxIdleConns
	transport.MaxIdleConnsPerHost = maxIdleConnsPerHost
	transport.IdleConnTimeout = idleConnTimeout

	return transport
}
