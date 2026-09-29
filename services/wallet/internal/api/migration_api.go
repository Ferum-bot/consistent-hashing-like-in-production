package api

import (
	"log/slog"
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/ndjson"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/request"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/response"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/models"
)

type MigrationAPI struct {
	migration MigrationService
}

func NewMigrationAPI(migration MigrationService) *MigrationAPI {
	return &MigrationAPI{migration: migration}
}

func (a *MigrationAPI) Export(w http.ResponseWriter, r *http.Request) {
	users, err := a.migration.Export(r.Context())
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	dtos := make([]userDTO, 0, len(users))

	for _, user := range users {
		dtos = append(dtos, toUserDTO(user))
	}

	if err := ndjson.Write(w, dtos); err != nil {
		slog.Warn("export interrupted", "err", err)
	}
}

func (a *MigrationAPI) Import(w http.ResponseWriter, r *http.Request) {
	dtos, err := ndjson.Decode[userDTO](r)
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	users := make([]models.User, 0, len(dtos))

	for _, dto := range dtos {
		users = append(users, dto.toModel())
	}

	if err := a.migration.Import(r.Context(), users); err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	response.JSON(w, http.StatusOK, importResponse{Imported: len(users)})
}

func (a *MigrationAPI) Evict(w http.ResponseWriter, r *http.Request) {
	var req evictRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	evicted, err := a.migration.Evict(r.Context(), req.UserIDs)
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	response.JSON(w, http.StatusOK, evictResponse{Evicted: evicted})
}
