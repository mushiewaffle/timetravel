package api

import (
	"github.com/gorilla/mux"
	"github.com/rainbowmga/timetravel/service"
)

// V2API provides /api/v2 endpoints for versioned record access.
type V2API struct {
	records service.VersionedRecordService
}

// NewV2API constructs a V2API.
func NewV2API(records service.VersionedRecordService) *V2API {
	return &V2API{records: records}
}

// CreateRoutes generates all /api/v2 routes.
func (a *V2API) CreateRoutes(routes *mux.Router) {
	routes.Path("/records/{id}").HandlerFunc(a.GetRecordV2).Methods("GET")
	routes.Path("/records/{id}").HandlerFunc(a.PostRecordV2).Methods("POST")
	routes.Path("/records/{id}/versions").HandlerFunc(a.ListRecordVersionsV2).Methods("GET")
}
