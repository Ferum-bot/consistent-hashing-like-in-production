package main

import "strings"

const (
	routersFlag = "routers"
	walletsFlag = "wallets"
	loadgenFlag = "loadgen"

	defaultLoadgenURL = "http://localhost:8090"
)

var (
	defaultRouterURLs = []string{
		"http://localhost:8081",
		"http://localhost:8082",
	}

	defaultWalletURLs = []string{
		"http://localhost:9001",
		"http://localhost:9002",
		"http://localhost:9003",
		"http://localhost:9004",
		"http://localhost:9005",
	}
)

type urlList []string

func (l *urlList) String() string {
	return strings.Join(*l, ",")
}

func (l *urlList) Set(value string) error {
	var urls []string

	for _, url := range strings.Split(value, ",") {
		if url = strings.TrimSpace(url); url != "" {
			urls = append(urls, url)
		}
	}

	*l = urls

	return nil
}
