package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
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

	// TODO check for unique team name

	// TODO create team entry

}

// output meaning
// -2: user not found
// -1: general error
// 0: no permission/not part of team
// 1: team member
// 2: manager
// 3: owner
func checkPermissionByEmail(email string, teamid int) int {
	ctx := context.Background()
	param := user.ListParams{EmailAddresses: []string{email}} // FIXME rename param => something useful

	// we expect only one user per email
	res, err := user.List(ctx, &param)
	if err != nil || len(res.Users) > 1 {
		return -1
	} else if len(res.Users) == 0 {
		return -2
	}

	userid := ""
	for _, user := range res.Users {
		userid = user.ID
	}

	permission := checkPermissionByID(teamid, userid)

	return permission
}

func checkPermissionByID(teamid int, userid string) int {
	db := createDB()
	defer db.Close()

	permission, err := db.Query("SELECT level FROM teampermission WHERE teamid = ? AND userid = ? ", teamid, userid)
	if err != nil {
		return -1
	}
	for permission.Next() {
		var level int
		permission.Scan(&level)
		return level
	}

	return -1
}

// input: email of person and what team
func getPermission(w http.ResponseWriter, r *http.Request) {
	_, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	if r.URL.Query().Get("userEmail") == "" && r.URL.Query().Get("teamId") == "" {
		fmt.Fprintf(w, `
		{
			"response": "incorrect format"
		}
		`)
		return
	}

	user := r.URL.Query().Get("userEmail")
	team := r.URL.Query().Get("teamId")
	teamid, err := strconv.Atoi(team)
	if err != nil {
		fmt.Fprintf(w, `
		{
			"response": "incorrect format"
		}
		`)
		return
	}
	level := checkPermissionByEmail(user, teamid) // TODO handle error?
	fmt.Fprintf(w, `
	{
		"response": "ok",
		"permission": "%d"
	}
	`, level)
}

// inputs: email of person to set, and the desired permission level, and what team
func setPermission(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	if r.URL.Query().Get("userEmail") == "" && r.URL.Query().Get("teamId") == "" {
		fmt.Fprintf(w, `
		{
			"response": "incorrect format"
		}
		`)
		return
	}
	// get the token's user id
	setterId := claims.Subject
	user := r.URL.Query().Get("userEmail") // the user to set a permission for
	team := r.URL.Query().Get("teamId")
	level := r.URL.Query().Get("permissionLevel")
	teamid, err := strconv.Atoi(team)
	if err != nil {
		fmt.Fprintf(w, `
		{
			"response": "incorrect format"
		}
		`)
		return
	}
	proposedPermission, err := strconv.Atoi(level)
	if err != nil {
		fmt.Fprintf(w, `
		{
			"response": "incorrect format"
		}
		`)
		return
	}

	setterPermission := checkPermissionByID(teamid, setterId)
	userPermisssion := checkPermissionByEmail(user, teamid)

	// check if user has permission to set permissions
	// if person to set has a higher permission level than user, error out, or if proposed permission is higher
	if setterPermission < 2 {
		fmt.Fprintf(w, `
		{
			"response": "Insufficient permission"
		}
		`)
		return
	} else if userPermisssion >= setterPermission {
		fmt.Fprintf(w, `
		{
			"response": "invalid permission"
		}
		`)
		return
	} else if proposedPermission >= setterPermission {
		fmt.Fprintf(w, `
		{
			"response": "Insufficient permission"
		}
		`)
		return
	}
	userID := getUserIDByEmail(user)
	// otherwise upsert teampermission
	db := createDB()
	defer db.Close()
	db.Exec("INSERT INTO teampermission(userid, teamid, level) VALUES(?, ?, ?) ON CONFLICT(userid, teamid) DO UPDATE SET level=?", userID, teamid, proposedPermission, proposedPermission)
	fmt.Fprintf(w, `
	{
		"response": "valid"
	}
	`)
}
