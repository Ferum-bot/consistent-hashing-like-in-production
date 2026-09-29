package commands

import (
	"errors"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/tools/chctl/client"
)

const (
	hintStartStand   = "the stand is not running, start a scenario from scenarios/"
	hintApplyFirst   = "routers have no topology yet, apply one: chctl apply <topology.json>"
	hintGenerateData = "generate the snapshot first with datagen"
	hintReloadLedger = "load the ledger once the snapshot exists: chctl load reload"
	hintAbort        = "finish it or roll it back: chctl abort"
	hintStatus       = "see what is running: chctl status"
	hintWalletURL    = "is a wallet running at this url? import uses " + defaultWalletURL + " unless another url is given"
)

type hintedError struct {
	err  error
	hint string
}

func (e *hintedError) Error() string {
	return e.err.Error()
}

func (e *hintedError) Unwrap() error {
	return e.err
}

func withHint(err error, hint string) error {
	if err == nil {
		return nil
	}

	return &hintedError{err: err, hint: hint}
}

func Explain(err error) string {
	if hint := hintFor(err); hint != "" {
		return err.Error() + "\nhint: " + hint
	}

	return err.Error()
}

func hintFor(err error) string {
	var hinted *hintedError
	if errors.As(err, &hinted) {
		return hinted.hint
	}

	var unreachable *client.UnreachableError
	if !errors.As(err, &unreachable) {
		return ""
	}

	if unreachable.Service == "wallet" {
		return hintWalletURL
	}

	return hintStartStand
}
