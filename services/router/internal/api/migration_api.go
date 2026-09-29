package api

import (
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/response"
)

type MigrationAPI struct {
	migration MigrationService
}

func NewMigrationAPI(migration MigrationService) *MigrationAPI {
	return &MigrationAPI{migration: migration}
}

func (a *MigrationAPI) Run(w http.ResponseWriter, r *http.Request) {
	result, err := a.migration.Run(r.Context())
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	response.JSON(w, http.StatusOK, migrationResponse{Users: result.Users, PerNode: result.PerNode})
}

func (a *MigrationAPI) Cleanup(w http.ResponseWriter, r *http.Request) {
	result, err := a.migration.Cleanup(r.Context())
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	response.JSON(w, http.StatusOK, cleanupResponse{Users: result.Users})
}
