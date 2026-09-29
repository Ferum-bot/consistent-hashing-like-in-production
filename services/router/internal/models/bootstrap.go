package models

// BootstrapResult — сколько пользователей из снапшота разложено по каким нодам.
type BootstrapResult struct {
	Users   int
	PerNode map[string]int
}
