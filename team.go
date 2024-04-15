package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/clerk/clerk-sdk-go/v2"
)

func createTeam(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	_ = claims // temp

	if os.Getenv("OPEN_TEAMS") == "0" {
		fmt.Fprintf(w, `
		{
			"status": "disabled"
		}`)
		return
	}

	// TODO check for unique team name?
}
