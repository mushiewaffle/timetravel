package entity

import "time"

// RecordVersion describes a persisted version of a record.
type RecordVersion struct {
	RecordID   int       `json:"record_id"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	DataDigest string    `json:"data_digest,omitempty"`
}

// VersionedRecord is the record payload along with version metadata.
type VersionedRecord struct {
	ID        int               `json:"id"`
	Version   int               `json:"version"`
	CreatedAt time.Time         `json:"created_at"`
	Data      map[string]string `json:"data"`
}
