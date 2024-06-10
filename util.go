package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/joshtenorio/glassypdm-server/sqlcgen"
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
		Key  string `json:"clerk_publickey"`
		Name string `json:"name"`
	}{}
	data.Key = os.Getenv("CLERK_PUBLICKEY")
	data.Name = os.Getenv("SERVER_NAME")
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

//go:embed schema.sql
var ddl string

func UseQueries() *sqlcgen.Queries {
	ctx := context.Background()
	db := createDB()
	defer db.Close()

	_, err := db.ExecContext(ctx, ddl)
	if err != nil {
		os.Exit(1)
	}

	queries := sqlcgen.New(db)
	return queries
}

func getUserIDByEmail(email string) string {
	ctx := context.Background()
	param := user.ListParams{EmailAddresses: []string{email}}

	// we expect only one user per email
	res, err := user.List(ctx, &param)
	// FIXME handle error
	if err != nil || len(res.Users) > 1 {
		return ""
	} else if len(res.Users) == 0 {
		return ""
	}

	userid := ""
	for _, user := range res.Users {
		userid = user.ID
	}
	return userid
}

// checks if a user can generally upload files
// doesn't check permission for specific projects/teams
func canUserUpload(userId string) bool {
	ctx := context.Background()
	query := UseQueries()

	// check team permission
	// TODO: ensure that a team has at least one project for this to be a valid check
	teampermissions, err := query.FindTeamPermissions(ctx, sql.NullString{String: userId, Valid: true})
	if err != nil {
		return false
	}
	for level := range teampermissions {
		if level >= 2 {
			return true
		}
	}

	// check project permission
	projectpermissions, err := query.FindProjectPermissions(ctx, sql.NullString{String: userId, Valid: true})
	if err != nil {
		return false
	}
	for level := range projectpermissions {
		if level >= 2 {
			return true
		}
	}
	return false
}
