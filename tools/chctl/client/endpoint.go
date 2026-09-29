package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	readyTimeout = time.Second
	quickTimeout = 5 * time.Second
	maxErrorBody = 4096
)

type request struct {
	method      string
	path        string
	body        []byte
	contentType string
	timeout     time.Duration
}

type endpoint struct {
	service string
	url     string
	http    *http.Client
}

func newEndpoint(service, url string, client *http.Client) endpoint {
	return endpoint{service: service, url: strings.TrimRight(url, "/"), http: client}
}

func (e endpoint) Service() string {
	return e.service
}

func (e endpoint) URL() string {
	return e.url
}

func (e endpoint) Ready(ctx context.Context) error {
	return e.do(ctx, request{method: http.MethodGet, path: "/healthz", timeout: readyTimeout}, nil)
}

func (e endpoint) get(ctx context.Context, path string, out any) error {
	return e.do(ctx, request{method: http.MethodGet, path: path, timeout: quickTimeout}, out)
}

func (e endpoint) send(ctx context.Context, method, path string, payload any, timeout time.Duration, out any) error {
	req := request{method: method, path: path, timeout: timeout}

	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		req.body = body
		req.contentType = "application/json"
	}

	return e.do(ctx, req, out)
}

func (e endpoint) do(ctx context.Context, req request, out any) error {
	ctx, cancel := context.WithTimeout(ctx, req.timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, req.method, e.url+req.path, bytes.NewReader(req.body))
	if err != nil {
		return err
	}

	if req.contentType != "" {
		httpReq.Header.Set("Content-Type", req.contentType)
	}

	resp, err := e.http.Do(httpReq)
	if err != nil {
		return &UnreachableError{Service: e.service, URL: e.url, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return e.rejected(resp)
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%s %s: unexpected answer: %w", e.service, e.url, err)
	}

	return nil
}

func (e endpoint) rejected(resp *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))

	var body struct {
		Error string `json:"error"`
	}

	message := strings.TrimSpace(string(data))

	if json.Unmarshal(data, &body) == nil && body.Error != "" {
		message = body.Error
	}

	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}

	return &ResponseError{Service: e.service, URL: e.url, Status: resp.StatusCode, Message: message}
}
