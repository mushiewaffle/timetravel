package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rainbowmga/timetravel/entity"

	_ "modernc.org/sqlite"
)

// SQLiteRecordService is a SQLite-backed implementation of RecordService and VersionedRecordService.
//
// Data model (simple, auditable):
// - Each record update is persisted as an immutable snapshot row.
// - A record's current state is the latest snapshot by version.
// - History/time-travel is supported by reading older snapshots.
type SQLiteRecordService struct {
	db *sql.DB
}

// NewSQLiteRecordService opens (or creates) a SQLite database and ensures schema exists.
func NewSQLiteRecordService(dbPath string) (*SQLiteRecordService, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	svc := &SQLiteRecordService{db: db}
	if err := svc.ensureSchema(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return svc, nil
}

// Close releases the underlying DB resources.
func (s *SQLiteRecordService) Close() error {
	return s.db.Close()
}

func (s *SQLiteRecordService) ensureSchema(ctx context.Context) error {
	// record_versions holds all record history.
	//
	// created_at is stored as an RFC3339Nano string for easy ordering/comparison.
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS record_versions (
			record_id INTEGER NOT NULL,
			version INTEGER NOT NULL,
			data_json TEXT NOT NULL,
			data_digest TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (record_id, version)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_record_versions_record_created_at ON record_versions(record_id, created_at);`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLiteRecordService) GetRecord(ctx context.Context, id int) (entity.Record, error) {
	// v1 semantics: return only the latest state (no version metadata).
	if id <= 0 {
		return entity.Record{}, ErrRecordIDInvalid
	}

	vr, err := s.getLatestVersion(ctx, id)
	if err != nil {
		return entity.Record{}, err
	}

	return entity.Record{ID: vr.ID, Data: vr.Data}, nil
}

func (s *SQLiteRecordService) CreateRecord(ctx context.Context, record entity.Record) error {
	// v1 create semantics: create the record if it doesn't exist.
	// This inserts version=1.
	id := record.ID
	if id <= 0 {
		return ErrRecordIDInvalid
	}

	exists, err := s.recordExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return ErrRecordAlreadyExists
	}

	dataJSON, err := encodeData(record.Data)
	if err != nil {
		return err
	}
	createdAt := time.Now().UTC()
	digest := snapshotDigest(dataJSON)

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO record_versions(record_id, version, data_json, data_digest, created_at) VALUES(?, 1, ?, ?, ?);`,
		id,
		string(dataJSON),
		digest,
		createdAt.Format(time.RFC3339Nano),
	)
	return err
}

func (s *SQLiteRecordService) UpdateRecord(ctx context.Context, id int, updates map[string]*string) (entity.Record, error) {
	// v1 update semantics: the record must already exist.
	// We write a new snapshot version and return the new latest state.
	if id <= 0 {
		return entity.Record{}, ErrRecordIDInvalid
	}

	vr, err := s.applyUpdate(ctx, id, updates, false)
	if err != nil {
		return entity.Record{}, err
	}

	return entity.Record{ID: vr.ID, Data: vr.Data}, nil
}

func (s *SQLiteRecordService) GetLatestRecord(ctx context.Context, id int) (entity.Record, error) {
	// v2 helper: explicitly named "latest".
	return s.GetRecord(ctx, id)
}

func (s *SQLiteRecordService) GetRecordVersion(ctx context.Context, id int, version int) (entity.Record, error) {
	// v2 explicit version read.
	if id <= 0 {
		return entity.Record{}, ErrRecordIDInvalid
	}
	if version <= 0 {
		return entity.Record{}, ErrRecordDoesNotExist
	}

	vr, err := s.getExplicitVersion(ctx, id, version)
	if err != nil {
		return entity.Record{}, err
	}

	return entity.Record{ID: vr.ID, Data: vr.Data}, nil
}

func (s *SQLiteRecordService) GetRecordAt(ctx context.Context, id int, at time.Time) (entity.Record, error) {
	// v2 time-travel read: returns the latest version whose created_at <= at.
	if id <= 0 {
		return entity.Record{}, ErrRecordIDInvalid
	}

	vr, err := s.getVersionAt(ctx, id, at.UTC())
	if err != nil {
		return entity.Record{}, err
	}

	return entity.Record{ID: vr.ID, Data: vr.Data}, nil
}

func (s *SQLiteRecordService) ListRecordVersions(ctx context.Context, id int) ([]entity.RecordVersion, error) {
	// v2 history listing.
	if id <= 0 {
		return nil, ErrRecordIDInvalid
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT record_id, version, created_at, data_digest FROM record_versions WHERE record_id = ? ORDER BY version ASC;`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := []entity.RecordVersion{}
	for rows.Next() {
		var recordID int
		var version int
		var createdAtRaw string
		var digest string
		if err := rows.Scan(&recordID, &version, &createdAtRaw, &digest); err != nil {
			return nil, err
		}
		createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
		if err != nil {
			return nil, err
		}
		versions = append(versions, entity.RecordVersion{RecordID: recordID, Version: version, CreatedAt: createdAt, DataDigest: digest})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, ErrRecordDoesNotExist
	}

	return versions, nil
}

func (s *SQLiteRecordService) ApplyUpdate(ctx context.Context, id int, updates map[string]*string) (entity.VersionedRecord, error) {
	// v2 update semantics: apply updates to the latest snapshot and create a new snapshot version.
	// If the record doesn't exist yet, this creates version=1.
	if id <= 0 {
		return entity.VersionedRecord{}, ErrRecordIDInvalid
	}
	return s.applyUpdate(ctx, id, updates, true)
}

func (s *SQLiteRecordService) applyUpdate(ctx context.Context, id int, updates map[string]*string, createIfMissing bool) (entity.VersionedRecord, error) {
	// Core write path.
	//
	// We use a transaction so that:
	// - we read the latest version and write the next version atomically
	// - concurrent writers won't accidentally produce duplicate versions
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return entity.VersionedRecord{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	latest, err := s.getLatestVersionTx(ctx, tx, id)
	if err != nil {
		if !errors.Is(err, ErrRecordDoesNotExist) {
			return entity.VersionedRecord{}, err
		}
		if !createIfMissing {
			return entity.VersionedRecord{}, err
		}

		// First write creates version 1 from the updates applied to an empty record.
		nextData := applyUpdates(map[string]string{}, updates)
		dataJSON, err := encodeData(nextData)
		if err != nil {
			return entity.VersionedRecord{}, err
		}
		createdAt := time.Now().UTC()
		digest := snapshotDigest(dataJSON)

		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO record_versions(record_id, version, data_json, data_digest, created_at) VALUES(?, 1, ?, ?, ?);`,
			id,
			string(dataJSON),
			digest,
			createdAt.Format(time.RFC3339Nano),
		)
		if err != nil {
			return entity.VersionedRecord{}, err
		}
		if err := tx.Commit(); err != nil {
			return entity.VersionedRecord{}, err
		}

		return entity.VersionedRecord{ID: id, Version: 1, CreatedAt: createdAt, Data: nextData}, nil
	}

	nextData := applyUpdates(latest.Data, updates)
	dataJSON, err := encodeData(nextData)
	if err != nil {
		return entity.VersionedRecord{}, err
	}
	createdAt := time.Now().UTC()
	digest := snapshotDigest(dataJSON)
	nextVersion := latest.Version + 1

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO record_versions(record_id, version, data_json, data_digest, created_at) VALUES(?, ?, ?, ?, ?);`,
		id,
		nextVersion,
		string(dataJSON),
		digest,
		createdAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return entity.VersionedRecord{}, err
	}

	if err := tx.Commit(); err != nil {
		return entity.VersionedRecord{}, err
	}

	return entity.VersionedRecord{ID: id, Version: nextVersion, CreatedAt: createdAt, Data: nextData}, nil
}

func (s *SQLiteRecordService) String() string {
	return "SQLiteRecordService"
}
