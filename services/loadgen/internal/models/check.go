package models

// CheckResult — чем закончилась сверка баланса пользователя с книгой.
type CheckResult string

const (
	// CheckOK — сервис вернул ровно то, что записано в книге.
	CheckOK CheckResult = "ok"
	// CheckMissing — сервис ответил 404, хотя пользователь существует.
	CheckMissing CheckResult = "missing"
	// CheckLost — на сервисе денег меньше, чем должно быть.
	CheckLost CheckResult = "lost"
	// CheckExtra — на сервисе денег больше, чем должно быть.
	CheckExtra CheckResult = "extra"
	// CheckSkipped — сверять нельзя: по пользователю шла запись или результат прошлой операции неизвестен.
	CheckSkipped CheckResult = "skipped"
)

// LedgerStats — что сейчас лежит в книге ожидаемых балансов.
type LedgerStats struct {
	Loaded  bool
	Users   int
	Wallets int
	Money   int64
	// Uncertain — пользователи, по которым операция оборвалась и неизвестно, применилась ли она.
	Uncertain int
}
