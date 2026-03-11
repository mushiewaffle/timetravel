package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rainbowmga/timetravel/entity"
)

func (s *SQLiteRecordService) recordExists(ctx context.Context, id int) (bool, error) {
	row := s.db.QueryRowContext(ctx, `SELECT 1 FROM record_versions WHERE record_id = ? LIMIT 1;`, id)
	var one int
	if err := row.Scan(&one); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *SQLiteRecordService) getLatestVersion(ctx context.Context, id int) (entity.VersionedRecord, error) {
	return s.getLatestVersionTx(ctx, nil, id)
}

func (s *SQLiteRecordService) getLatestVersionTx(ctx context.Context, tx *sql.Tx, id int) (entity.VersionedRecord, error) {
	query := `SELECT record_id, version, data_json, created_at FROM record_versions WHERE record_id = ? ORDER BY version DESC LIMIT 1;`
	var row *sql.Row
	if tx != nil {
		row = tx.QueryRowContext(ctx, query, id)
	} else {
		row = s.db.QueryRowContext(ctx, query, id)
	}

	var recordID int
	var version int
	var dataJSONRaw string
	var createdAtRaw string
	if err := row.Scan(&recordID, &version, &dataJSONRaw, &createdAtRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.VersionedRecord{}, ErrRecordDoesNotExist
		}
		return entity.VersionedRecord{}, err
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return entity.VersionedRecord{}, err
	}
	data, err := decodeData([]byte(dataJSONRaw))
	if err != nil {
		return entity.VersionedRecord{}, err
	}

	return entity.VersionedRecord{ID: recordID, Version: version, CreatedAt: createdAt, Data: data}, nil
}

func (s *SQLiteRecordService) getExplicitVersion(ctx context.Context, id int, version int) (entity.VersionedRecord, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT record_id, version, data_json, created_at FROM record_versions WHERE record_id = ? AND version = ? LIMIT 1;`,
		id,
		version,
	)

	var recordID int
	var ver int
	var dataJSONRaw string
	var createdAtRaw string
	if err := row.Scan(&recordID, &ver, &dataJSONRaw, &createdAtRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.VersionedRecord{}, ErrRecordDoesNotExist
		}
		return entity.VersionedRecord{}, err
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return entity.VersionedRecord{}, err
	}
	data, err := decodeData([]byte(dataJSONRaw))
	if err != nil {
		return entity.VersionedRecord{}, err
	}

	return entity.VersionedRecord{ID: recordID, Version: ver, CreatedAt: createdAt, Data: data}, nil
}

func (s *SQLiteRecordService) getVersionAt(ctx context.Context, id int, at time.Time) (entity.VersionedRecord, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT record_id, version, data_json, created_at FROM record_versions WHERE record_id = ? AND created_at <= ? ORDER BY version DESC LIMIT 1;`,
		id,
		at.Format(time.RFC3339Nano),
	)

	var recordID int
	var ver int
	var dataJSONRaw string
	var createdAtRaw string
	if err := row.Scan(&recordID, &ver, &dataJSONRaw, &createdAtRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.VersionedRecord{}, ErrRecordDoesNotExist
		}
		return entity.VersionedRecord{}, err
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return entity.VersionedRecord{}, err
	}
	data, err := decodeData([]byte(dataJSONRaw))
	if err != nil {
		return entity.VersionedRecord{}, err
	}

	return entity.VersionedRecord{ID: recordID, Version: ver, CreatedAt: createdAt, Data: data}, nil
}
