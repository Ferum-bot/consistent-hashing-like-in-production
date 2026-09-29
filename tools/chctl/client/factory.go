package client

import "net/http"

type Factory struct {
	http *http.Client
}

func NewFactory(client *http.Client) Factory {
	return Factory{http: client}
}

func (f Factory) Router(url string) *Router {
	return NewRouter(url, f.http)
}

func (f Factory) Wallet(url string) *Wallet {
	return NewWallet(url, f.http)
}

func (f Factory) Loadgen(url string) *Loadgen {
	return NewLoadgen(url, f.http)
}
