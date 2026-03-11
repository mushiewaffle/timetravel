package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func snapshotDigest(dataJSON []byte) string {
	sum := sha256.Sum256(dataJSON)
	return hex.EncodeToString(sum[:])
}

func encodeData(data map[string]string) ([]byte, error) {
	if data == nil {
		data = map[string]string{}
	}
	return json.Marshal(data)
}

func decodeData(dataJSON []byte) (map[string]string, error) {
	var data map[string]string
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return nil, err
	}
	if data == nil {
		data = map[string]string{}
	}
	return data, nil
}

func applyUpdates(current map[string]string, updates map[string]*string) map[string]string {
	next := map[string]string{}
	for k, v := range current {
		next[k] = v
	}
	for key, value := range updates {
		if value == nil {
			delete(next, key)
			continue
		}
		next[key] = *value
	}
	return next
}
