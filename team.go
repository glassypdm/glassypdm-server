package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/go-chi/chi/v5"
	"github.com/joshtenorio/glassypdm-server/sqlcgen"
)

type Member struct {
	EmailID string `json:"emailID"`
	Name    string `json:"name"`
	Role    string `json:"role"`
}

type TeamCreationRequest struct {
	Name string `json:"name"`
}

func createTeam(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	if os.Getenv("OPEN_TEAMS") == "0" {
		fmt.Fprintf(w, `{ "status": "disabled" }`)
		return
	}
	query := UseQueries()

	var request TeamCreationRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		fmt.Fprintf(w, `{ "status": "json bad" }`)
		return
	}

	// create team entry and add user as owner
	id, err := query.InsertTeam(ctx, request.Name)
	if err != nil {
		if strings.Contains(strings.Split(err.Error(), "SQLite error: ")[1], "UNIQUE constraint failed") {
			fmt.Fprintf(w, `{ "status": "error", "message": "team name exists already" }`)
		} else {
			fmt.Fprintf(w, `{ "status": "error", "message": "db error" }`)
		}
		return
	}

	_, err = query.SetTeamPermission(ctx, sqlcgen.SetTeamPermissionParams{Teamid: id, Userid: claims.Subject, Level: 3})
	fmt.Println(id)
	fmt.Println(err)

	fmt.Fprintf(w, `{ "status": "success" }`)
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
	searchResult := user.ListParams{EmailAddresses: []string{email}}

	// we expect only one user per email
	res, err := user.List(ctx, &searchResult)
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
	ctx := context.Background()

	query := UseQueries()

	permission, err := query.GetTeamPermission(ctx, sqlcgen.GetTeamPermissionParams{Teamid: int64(teamid), Userid: userid})
	if err != nil {
		return -1
	}
	return int(permission)
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
		fmt.Fprintf(w, `{ "response": "incorrect format" }`)
		return
	}

	user := r.URL.Query().Get("userEmail")
	team := r.URL.Query().Get("teamId")
	teamid, err := strconv.Atoi(team)
	if err != nil {
		fmt.Fprintf(w, `{ "response": "incorrect format" }`)
		return
	}
	level := checkPermissionByEmail(user, teamid)
	fmt.Fprintf(w, `
	{
		"response": "ok",
		"permission": %d
	}
	`, level)
}

type PermissionRequest struct {
	Email  string `json:"email"`
	TeamId int    `json:"teamId"`
	Level  int    `json:"level"`
}

// inputs: email of person to set, and the desired permission level, and what team
// TODO: if setting a new owner, demote the old owner to manager
func setPermission(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	// get the token's user id
	setterId := claims.Subject
	var req PermissionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Println(err)
		fmt.Fprintf(w, `{ "status": "incorrect format" }`)
		return
	}
	user := req.Email // the user to set a permission for
	teamId := req.TeamId
	proposedPermission := req.Level

	setterPermission := checkPermissionByID(teamId, setterId)
	userPermisssion := checkPermissionByEmail(user, teamId)

	// check if user has permission to set permissions
	// if person to set has a higher permission level than user, error out, or if proposed permission is higher
	if setterPermission < 2 {
		fmt.Fprintf(w, ` { "status": "Insufficient permission" }`)
		return
	} else if userPermisssion >= setterPermission {
		fmt.Fprintf(w, `{ "status": "invalid permission" }`)
		return
	} else if proposedPermission >= setterPermission {
		fmt.Fprintf(w, `{ "status": "Insufficient permission" }`)
		return
	}
	userID := getUserIDByEmail(user)

	// otherwise upsert teampermission
	// TODO handle errors
	query := UseQueries()
	if proposedPermission != -4 {
		_, err = query.SetTeamPermission(ctx, sqlcgen.SetTeamPermissionParams{Userid: userID, Teamid: int64(teamId), Level: int64(proposedPermission)})

	} else {
		_, err = query.DeleteTeamPermission(ctx, userID)
	}
	if err != nil {
		fmt.Fprintf(w, `{ "status": "db" }`)
		fmt.Println(err)
		return
	}
	fmt.Fprintf(w, `{ "status": "valid" }`)
}

func getTeamForUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	queries := UseQueries()
	teams, err := queries.FindUserTeams(ctx, claims.Subject)
	if err != nil {
		fmt.Fprintf(w, `{ "status": "db error" }`)
		return
	}

	fmt.Println(teams)
	var output []Team

	for _, row := range teams {
		output = append(output, Team{Id: int(row.Teamid), Name: row.Name})
	}
	output_str, err := json.Marshal(output)
	if err != nil {
		return
	}
	fmt.Fprintf(w, `{ "open": %v, "teams": %s}`, IsServerOpen(), (output_str))
}

func getTeamInformation(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	userId := claims.Subject
	teamIdStr := chi.URLParam(r, "teamId")
	teamId, err := strconv.Atoi(teamIdStr)
	if err != nil {
		fmt.Fprintf(w, `{ "status": "incorrect format" }`)
		return
	}

	query := UseQueries()
	// check if team exists
	name, err := query.GetTeamName(ctx, int64(teamId))
	if err != nil {
		fmt.Fprintf(w, `{ "status": "team DNE" }`)
		return
	}

	level := checkPermissionByID(teamId, userId)
	levelStr := ""
	// if level is negative, you are not in the team
	// and do not have permission to see team membership
	if level < 0 {
		fmt.Fprintf(w, `{ "status": "no permission" }`)
		return
	}

	switch level {
	case 1:
		levelStr = "Member"
	case 2:
		levelStr = "Manager"
	case 3:
		levelStr = "Owner"
	default:
		levelStr = "Undefined"
	}

	memberdto, err := query.GetTeamMembership(ctx, int64(teamId))
	if err != nil {
		fmt.Fprintf(w, `{ "status": "db error" }`)
		return
	}
	var members []Member
	for _, member := range memberdto {
		var m Member

		switch member.Level {
		case 1:
			m.Role = "Member"
		case 2:
			m.Role = "Manager"
		case 3:
			m.Role = "Owner"
		default:
			m.Role = "Undefined"
		}

		usr, err := user.Get(ctx, member.Userid)
		if err != nil {
			fmt.Println("status: invalid user id")
			continue
		}
		m.Name = *usr.FirstName + " " + *usr.LastName
		// we don't send the actual email address
		// for slightly better security
		m.EmailID = *usr.PrimaryEmailAddressID
		members = append(members, m)
	}

	m, err := json.Marshal(members)
	if err != nil {
		fmt.Fprintf(w, `{ "status": "json error" }`)
		return
	}
	fmt.Fprintf(w, `
	{
		"status": "ok",
		"teamName": "%s",
		"role":"%s",
		"members": %s
	}`, name, levelStr, string(m))
}
