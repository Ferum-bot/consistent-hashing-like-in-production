package request

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func PathInt64(r *http.Request, name string) (int64, error) {
	raw := r.PathValue(name)

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, NewErrNotANumber(name, raw)
	}

	return value, nil
}

func HeaderInt64(r *http.Request, name string) (int64, bool, error) {
	raw := r.Header.Get(name)
	if raw == "" {
		return 0, false, nil
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, true, NewErrNotANumber(name, raw)
	}

	return value, true, nil
}

func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return NewErrDecodeBody(err)
	}

	return nil
}
