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
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
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
		// TODO ???
		WriteDefaultSuccess(w, "disabled")
		return
	}
	var request TeamCreationRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		WriteError(w, "json bad")
		return
	}

	// create team entry and add user as owner
	id, err := queries.InsertTeam(ctx, request.Name)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			log.Warn("team name exists already", "requested name", request.Name)
			WriteError(w, "team name exists already")
			return
		}
		log.Error("unhandled db error when creating team", "db", err)
		WriteError(w, "db error")
		return
	}

	_, err = queries.SetTeamPermission(ctx, sqlcgen.SetTeamPermissionParams{Teamid: id, Userid: claims.Subject, Level: 3})
	if err != nil {
		log.Error("couldn't insert owner permission", "err", err.Error(), "teamID", id, "userID", claims.Subject)
	}

	WriteDefaultSuccess(w, "team created")
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

	permission, err := queries.GetTeamPermission(ctx, sqlcgen.GetTeamPermissionParams{Teamid: int32(teamid), Userid: userid})
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
		WriteError(w, "incorrect format")
		return
	}
	user := req.Email // the user to set a permission for
	teamId := req.TeamId
	proposedPermission := req.Level

	setterPermission := checkPermissionByID(teamId, setterId)
	userPermisssion := checkPermissionByEmail(user, teamId)
	if userPermisssion == -2 {
		WriteError(w, "user does not exist")
		return
	} else if userPermisssion == -1 || setterPermission == -1 {
		WriteError(w, "generic error")
		return
	}

	// check if user has permission to set permissions
	// if person to set has a higher permission level than user, error out, or if proposed permission is higher
	if setterPermission < 2 {
		WriteError(w, "insufficient permission")
		return
	} else if userPermisssion >= setterPermission {
		WriteError(w, "invalid permission")
		return
	} else if proposedPermission >= setterPermission {
		WriteError(w, "insufficient permission")
		return
	}
	userID := getUserIDByEmail(user)

	// otherwise upsert teampermission
	// TODO handle errors
	if proposedPermission != -4 {
		_, err = queries.SetTeamPermission(ctx, sqlcgen.SetTeamPermissionParams{Userid: userID, Teamid: int32(teamId), Level: int32(proposedPermission)})

	} else {
		_, err = queries.DeleteTeamPermission(ctx, userID)
	}
	if err != nil {
		log.Error("couldn't edit team permission", "userid", userID, "team", teamId, "level", proposedPermission, "error", err.Error())
		WriteError(w, "db error")
		return
	}
	WriteDefaultSuccess(w, "valid")
}

func GetTeamForUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	Teams, err := queries.FindUserTeams(ctx, claims.Subject)
	if err != nil {
		log.Error("couldn't get user's teams", "user", claims.Subject, "err", err.Error())
		WriteError(w, "db error")
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
		WriteError(w, "couldn't create json")
		return
	}
	WriteSuccess(w, string(OutputBytes))
}

type GetTeamForUserResponse struct {
	Open  bool   `json:"open"`
	Teams []Team `json:"teams"`
}

func getTeamInformationByName(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	userId := claims.Subject
	teamName := chi.URLParam(r, "team-name")

	if teamName == "" {
		WriteError(w, "incorrect format")
		return
	}

	teamid, err := queries.GetTeamFromName(ctx, teamName)
	if err != nil {
		WriteError(w, "team not found")
		return
	}

	QueryTeamInformation(w, int(teamid), userId)
}

func getTeamInformation(w http.ResponseWriter, r *http.Request) {
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
		fmt.Fprintf(w, `{ "response": "incorrect format" }`)
		return
	}

	QueryTeamInformation(w, teamId, userId)

}

func QueryTeamInformation(w http.ResponseWriter, teamId int, userId string) {
	ctx := context.Background()
	// check if team exists
	name, err := queries.GetTeamName(ctx, int32(teamId))
	if err != nil {
		fmt.Fprintf(w, `{ "response": "team DNE" }`)
		return
	}

	level := checkPermissionByID(teamId, userId)
	levelStr := ""
	// if level is negative, you are not in the team
	// and do not have permission to see team membership
	if level < 0 {
		fmt.Fprintf(w, `{ "response": "no permission" }`)
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

	memberdto, err := queries.GetTeamMembership(ctx, int32(teamId))
	if err != nil {

		fmt.Fprintf(w, `{ "response": "db error" }`)
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
		/* POSTPONED  TODO
		email, err := emailaddress.Get(ctx, *usr.PrimaryEmailAddressID)
		if err != nil {
			log.Warn("couldn't get email from clerk", "emailID", *usr.PrimaryEmailAddressID)
			m.Email = ""
		} else {
			m.Email = email.EmailAddress
		}
		*/
		m.Email = ""
		members = append(members, m)
	}
	log.Info("found members for team", len(members))

	output := TeamInformation{
		TeamName: name,
		TeamId:   teamId,
		Role:     levelStr,
		Members:  members,
	}
	output_bytes, _ := json.Marshal(output)
	WriteSuccess(w, string(output_bytes))
}

type TeamInformation struct {
	TeamName string   `json:"team_name"`
	TeamId   int      `json:"team_id"`
	Role     string   `json:"role"`
	Members  []Member `json:"members"`
}

func GetBasicTeamInfo(w http.ResponseWriter, r *http.Request) {
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
		fmt.Fprintf(w, `{ "response": "incorrect format" }`)
		return
	}

	// check if team exists
	name, err := queries.GetTeamName(ctx, int32(teamId))
	if err != nil {
		fmt.Fprintf(w, `{ "response": "team DNE" }`)
		return
	}

	level := checkPermissionByID(teamId, userId)
	levelStr := ""
	// if level is negative, you are not in the team
	// and do not have permission to see team membership
	if level < 0 {
		fmt.Fprintf(w, `{ "response": "no permission" }`)
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

	output := TeamInformation{
		TeamName: name,
		Role:     levelStr,
		Members:  []Member{},
	}
	output_bytes, _ := json.Marshal(output)
	WriteSuccess(w, string(output_bytes))
}
