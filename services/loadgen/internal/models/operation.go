package models

import "time"

type OperationKind string

const (
	OperationRead     OperationKind = "read"
	OperationDeposit  OperationKind = "deposit"
	OperationWithdraw OperationKind = "withdraw"
	// OperationVerify — перечитывание недавно изменённого пользователя ради сверки баланса.
	OperationVerify OperationKind = "verify"
)

// Operation — одна операция клиента, которую надо отправить в роутер.
type Operation struct {
	Kind     OperationKind
	UserID   int64
	Currency string
	Amount   int64
	// ScheduledAt — когда операция должна была уйти. От этого момента меряется время ответа,
	// поэтому очередь внутри loadgen тоже попадает в задержку.
	ScheduledAt time.Time
}

// Wallet — кошелёк так, как его видит loadgen: валюта и баланс.
type Wallet struct {
	Currency string
	Balance  int64
}

// UserResponse — что роутер ответил на чтение пользователя.
type UserResponse struct {
	Status  int
	Wallets []Wallet
}
