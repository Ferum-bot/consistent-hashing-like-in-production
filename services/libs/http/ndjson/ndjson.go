package ndjson

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/request"
)

const ContentType = "application/x-ndjson"

func Write[T any](w http.ResponseWriter, items []T) error {
	w.Header().Set("Content-Type", ContentType)

	enc := json.NewEncoder(w)

	for _, item := range items {
		if err := enc.Encode(item); err != nil {
			return err
		}
	}

	return nil
}

func Decode[T any](r *http.Request) ([]T, error) {
	var items []T

	dec := json.NewDecoder(r.Body)

	for {
		var item T

		err := dec.Decode(&item)
		if errors.Is(err, io.EOF) {
			return items, nil
		}

		if err != nil {
			return nil, request.NewErrDecodeLine(len(items)+1, err)
		}

		items = append(items, item)
	}
}
