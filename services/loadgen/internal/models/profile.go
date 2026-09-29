package models

// Profile — как loadgen нагружает кластер. Меняется на лету через PATCH /config.
type Profile struct {
	// Running — идёт ли нагрузка. После старта loadgen стоит на паузе.
	Running bool
	// RPS — сколько операций в секунду отправлять.
	RPS int
	// ReadRatio — доля чтений среди всех операций, остальное — записи.
	ReadRatio float64
	// DepositShare — доля пополнений среди записей, остальное — списания.
	DepositShare float64
	// ReadZipfS и WriteZipfS — перекос выбора пользователя по закону Ципфа.
	// 0 — все пользователи равновероятны, больше 1 — немногие пользователи получают большую часть операций.
	ReadZipfS  float64
	WriteZipfS float64
	// HotShare — доля записей, которые уходят одному горячему пользователю-мерчанту.
	HotShare       float64
	MerchantUserID int64
	// MaxAmount — верхняя граница суммы одной операции в минимальных единицах валюты.
	MaxAmount int64
	// VerifyRPS — сколько раз в секунду перепроверять недавно изменённых пользователей.
	VerifyRPS int
}
