package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/headers"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/ndjson"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/snapshot"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/wait"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

const (
	maxErrorBody = 1024
	maxAttempts  = 3
	retryPause   = 200 * time.Millisecond
)

type WalletClient struct {
	http *http.Client
}

func NewWalletClient(client *http.Client) *WalletClient {
	return &WalletClient{http: client}
}

func (c *WalletClient) Import(ctx context.Context, node sharding.Node, version int64, body []byte) error {
	resp, err := c.do(ctx, node, http.MethodPost, "/import", version, ndjson.ContentType, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return rejected(node, resp)
	}

	return nil
}

func (c *WalletClient) Export(ctx context.Context, node sharding.Node, version int64, visit func(line []byte) error) error {
	resp, err := c.do(ctx, node, http.MethodGet, "/export", version, "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return rejected(node, resp)
	}

	scanner := snapshot.NewScanner(resp.Body)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		if err := visit(line); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return models.NewErrNodeUnavailable(node.ID, err)
	}

	return nil
}

func (c *WalletClient) Evict(ctx context.Context, node sharding.Node, version int64, userIDs []int64) error {
	return c.postJSON(ctx, node, "/evict", version, struct {
		UserIDs []int64 `json:"user_ids"`
	}{UserIDs: userIDs})
}

func (c *WalletClient) Fence(ctx context.Context, node sharding.Node, version int64, strict bool) error {
	return c.postJSON(ctx, node, "/fence", version, struct {
		Strict bool `json:"strict"`
	}{Strict: strict})
}

func (c *WalletClient) postJSON(ctx context.Context, node sharding.Node, path string, version int64, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := c.do(ctx, node, http.MethodPost, path, version, "application/json", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return rejected(node, resp)
	}

	return nil
}

func (c *WalletClient) do(
	ctx context.Context,
	node sharding.Node,
	method string,
	path string,
	version int64,
	contentType string,
	body []byte,
) (*http.Response, error) {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, node.Addr+path, bytes.NewReader(body))
		if err != nil {
			return nil, models.NewErrNodeUnavailable(node.ID, err)
		}

		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		setVersion(req, version)

		resp, err := c.http.Do(req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		c.http.CloseIdleConnections()

		if !wait.For(ctx, retryPause) {
			break
		}
	}

	return nil, models.NewErrNodeUnavailable(node.ID, lastErr)
}

func setVersion(req *http.Request, version int64) {
	if version > 0 {
		req.Header.Set(headers.TopologyVersion, strconv.FormatInt(version, 10))
	}
}

func rejected(node sharding.Node, resp *http.Response) error {
	answer, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	reason := errorMessage(answer)

	if resp.StatusCode == http.StatusConflict {
		return models.NewErrNodeFenced(node.ID, reason)
	}

	return models.NewErrNodeRejected(node.ID, resp.StatusCode, reason)
}

func errorMessage(answer []byte) string {
	var body struct {
		Error string `json:"error"`
	}

	if json.Unmarshal(answer, &body) == nil && body.Error != "" {
		return body.Error
	}

	return string(bytes.TrimSpace(answer))
}
