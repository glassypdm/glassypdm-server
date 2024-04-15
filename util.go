package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"database/sql"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func getVersion(w http.ResponseWriter, r *http.Request) {
	fmt.Println("brr")
	data := struct {
		Version string `json:"version"`
	}{}
	data.Version = os.Getenv("CLIENT_VERSION")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)

}

func getConfig(w http.ResponseWriter, r *http.Request) {
	fmt.Println("brr")
	data := struct {
		Key string `json:"clerk_publickey"`
	}{}
	data.Key = os.Getenv("CLERK_PUBLICKEY")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func createDB() *sql.DB {
	dburl := os.Getenv("TURSO_DATABASE_URL") + "?authToken=" + os.Getenv("TURSO_AUTH_TOKEN")
	db, err := sql.Open("libsql", dburl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open db %s: %s", dburl, err)
		os.Exit(1)
	}
	return db
}
