package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// GET /records/{id}/versions
func (a *V2API) ListRecordVersionsV2(w http.ResponseWriter, r *http.Request) {
	// Returns a list of available snapshots for the record.
	// This is primarily used by clients to discover which versions can be queried.
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	idNumber, err := strconv.ParseInt(id, 10, 32)
	if err != nil || idNumber <= 0 {
		err := writeError(w, "invalid id; id must be a positive number", http.StatusBadRequest)
		logError(err)
		return
	}

	versions, err := a.records.ListRecordVersions(ctx, int(idNumber))
	if err != nil {
		err := writeError(w, fmt.Sprintf("record of id %v does not exist", idNumber), http.StatusBadRequest)
		logError(err)
		return
	}

	err = writeJSON(w, versions, http.StatusOK)
	logError(err)
}
