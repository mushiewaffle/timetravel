package service

import (
	"context"
	"time"

	"github.com/rainbowmga/timetravel/entity"
)

// VersionedRecordService exposes record history and time travel queries.
//
// This is used by the /api/v2 endpoints.
type VersionedRecordService interface {
	// GetLatestRecord retrieves the latest version of a record.
	GetLatestRecord(ctx context.Context, id int) (entity.Record, error)

	// GetRecordVersion retrieves a record at an explicit version.
	GetRecordVersion(ctx context.Context, id int, version int) (entity.Record, error)

	// GetRecordAt retrieves the record state as-of the provided time (inclusive).
	GetRecordAt(ctx context.Context, id int, at time.Time) (entity.Record, error)

	// ListRecordVersions lists all versions of a record, ordered by version ascending.
	ListRecordVersions(ctx context.Context, id int) ([]entity.RecordVersion, error)

	// ApplyUpdate applies updates to the latest version of the record and persists a new version.
	ApplyUpdate(ctx context.Context, id int, updates map[string]*string) (entity.VersionedRecord, error)
}
