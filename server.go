package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/rainbowmga/timetravel/api"
	"github.com/rainbowmga/timetravel/service"
)

// logError logs all non-nil errors
func logError(err error) {
	if err != nil {
		log.Printf("error: %v", err)
	}
}

func main() {
	router := mux.NewRouter()

	dbPath := os.Getenv("TIMETRAVEL_DB_PATH")
	if dbPath == "" {
		dbPath = "timetravel.db"
	}

	records, err := service.NewSQLiteRecordService(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		logError(records.Close())
	}()

	v1api := api.NewAPI(records)
	v2api := api.NewV2API(records)

	apiRoute := router.PathPrefix("/api/v1").Subrouter()
	apiRoute.Path("/health").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		logError(err)
	})
	v1api.CreateRoutes(apiRoute)

	v2Route := router.PathPrefix("/api/v2").Subrouter()
	v2api.CreateRoutes(v2Route)

	address := "127.0.0.1:8000"
	srv := &http.Server{
		Handler:      router,
		Addr:         address,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("listening on %s", address)
	log.Fatal(srv.ListenAndServe())
}
