package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/go-chi/chi/v5"
	"github.com/joshtenorio/glassypdm-server/sqlcgen"
)

type TeamRole int

const (
	TeamRoleMember  = 1
	TeamRoleManager = 2
	TeamRoleOwner   = 3
)

func (tr TeamRole) EnumIndex() int {
	return int(tr)
}

type Member struct {
	EmailID string `json:"emailID"`
	Name    string `json:"name"`
	Role    string `json:"role"`
}

type TeamCreationRequest struct {
	Name string `json:"name"`
}

func CreateTeam(w http.ResponseWriter, r *http.Request) {
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
	if err != nil {
		log.Error("couldn't insert owner permission", "err", err.Error(), "teamID", id, "userID", claims.Subject)
	}

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
		log.Error("couldn't retrieve team permission", "team", teamid, "user", userid, "err", err.Error())
		return 0
	}
	return int(permission)
}

// input: email of person and what team
func GetPermission(w http.ResponseWriter, r *http.Request) {
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
	TeamId int    `json:"team_id"`
	Level  int    `json:"level"`
}

// inputs: email of person to set, and the desired permission level, and what team
// TODO: if setting a new owner, demote the old owner to manager
func SetPermission(w http.ResponseWriter, r *http.Request) {
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
		PrintError(w, "incorrect format")
		return
	}
	user := req.Email // the user to set a permission for
	teamId := req.TeamId
	proposedPermission := req.Level

	setterPermission := checkPermissionByID(teamId, setterId)
	userPermisssion := checkPermissionByEmail(user, teamId)
	if userPermisssion == -2 {
		PrintError(w, "user does not exist")
		return
	} else if userPermisssion == -1 || setterPermission == -1 {
		PrintError(w, "generic error")
		return
	}

	// check if user has permission to set permissions
	// if person to set has a higher permission level than user, error out, or if proposed permission is higher
	if setterPermission < 2 {
		PrintError(w, "insufficient permission")
		return
	} else if userPermisssion >= setterPermission {
		PrintError(w, "invalid permission")
		return
	} else if proposedPermission >= setterPermission {
		PrintError(w, "insufficient permission")
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
		log.Error("couldn't edit team permission", "userid", userID, "team", teamId, "level", proposedPermission, "error", err.Error())
		PrintError(w, "db error")
		return
	}
	PrintSuccess(w, "\"valid\"")
}

func GetTeamForUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	queries := UseQueries()
	Teams, err := queries.FindUserTeams(ctx, claims.Subject)
	if err != nil {
		log.Error("couldn't get user's teams", "user", claims.Subject, "err", err.Error())
		PrintError(w, "db error")
		return
	}

	var Output GetTeamForUserResponse
	var TeamList []Team

	for _, row := range Teams {
		TeamList = append(TeamList, Team{Id: int(row.Teamid), Name: row.Name})
	}

	Output.Open = IsServerOpen()
	Output.Teams = TeamList

	OutputBytes, err := json.Marshal(Output)
	if err != nil {
		PrintError(w, "couldn't create json")
		return
	}
	PrintSuccess(w, string(OutputBytes))
}

type GetTeamForUserResponse struct {
	Open  bool   `json:"open"`
	Teams []Team `json:"teams"`
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
	teamIdStr := chi.URLParam(r, "team-id")
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
			log.Warn("userid not found in clerk", "user", member.Userid)
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
