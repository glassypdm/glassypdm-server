package main

import (
	"net/http"
	"strconv"

	"github.com/clerk/clerk-sdk-go/v2"
)

// TODO
// body: filepath, project id
// returns commit # and hash list
func getLatestRevision(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	pid, err := strconv.Atoi(r.URL.Query().Get("pid"))
	if err != nil {
		// TODO return error, bad format
	}
	// TODO get teamid from pid
	if getProjectPermissionByID(claims.Subject, pid, 0) < 1 {
		// TODO no read permission so return error
	}

	db := createDB()
	defer db.Close()

	// TODO get most up-to-date file revision for filepath
	//_, _ := db.Query("SELECT * FROM ")
}

// TODO
// function that returns list of new files since commit number
// body: project id
func getNewFiles(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
}
