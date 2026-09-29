package snapshot

import (
	"bufio"
	"encoding/json"
	"io"
)

const (
	initialLineSize = 64 * 1024
	maxLineSize     = 1 << 20
)

func NewScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, initialLineSize), maxLineSize)

	return scanner
}

func UserID(line []byte) (int64, error) {
	var head struct {
		UserID int64 `json:"user_id"`
	}

	if err := json.Unmarshal(line, &head); err != nil {
		return 0, err
	}

	return head.UserID, nil
}
