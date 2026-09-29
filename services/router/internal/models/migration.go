package models

// MigrationResult — сколько пользователей скопировано на новые ноды и на какие.
type MigrationResult struct {
	Users int
	// PerNode — сколько пользователей получила каждая нода.
	PerNode map[string]int
}

// CleanupResult — сколько перенесённых пользователей удалено со старых нод.
type CleanupResult struct {
	Users int
}
