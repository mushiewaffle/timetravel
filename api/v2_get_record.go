package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// GET /records/{id}?version={n}&at={rfc3339}
func (a *V2API) GetRecordV2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	idNumber, err := strconv.ParseInt(id, 10, 32)
	if err != nil || idNumber <= 0 {
		err := writeError(w, "invalid id; id must be a positive number", http.StatusBadRequest)
		logError(err)
		return
	}

	q := r.URL.Query()
	versionStr := q.Get("version")
	atStr := q.Get("at")

	if versionStr != "" {
		ver, err := strconv.ParseInt(versionStr, 10, 32)
		if err != nil || ver <= 0 {
			err := writeError(w, "invalid version; version must be a positive number", http.StatusBadRequest)
			logError(err)
			return
		}
		rec, err := a.records.GetRecordVersion(ctx, int(idNumber), int(ver))
		if err != nil {
			err := writeError(w, fmt.Sprintf("record of id %v does not exist", idNumber), http.StatusBadRequest)
			logError(err)
			return
		}
		err = writeJSON(w, rec, http.StatusOK)
		logError(err)
		return
	}

	if atStr != "" {
		at, err := time.Parse(time.RFC3339Nano, atStr)
		if err != nil {
			err := writeError(w, "invalid at; must be an RFC3339 timestamp", http.StatusBadRequest)
			logError(err)
			return
		}
		rec, err := a.records.GetRecordAt(ctx, int(idNumber), at)
		if err != nil {
			err := writeError(w, fmt.Sprintf("record of id %v does not exist", idNumber), http.StatusBadRequest)
			logError(err)
			return
		}
		err = writeJSON(w, rec, http.StatusOK)
		logError(err)
		return
	}

	rec, err := a.records.GetLatestRecord(ctx, int(idNumber))
	if err != nil {
		err := writeError(w, fmt.Sprintf("record of id %v does not exist", idNumber), http.StatusBadRequest)
		logError(err)
		return
	}

	err = writeJSON(w, rec, http.StatusOK)
	logError(err)
}
