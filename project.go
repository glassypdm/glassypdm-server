package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
)

type Project struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Team string `json:"team"`
}

// TODO move this to team.go?
type Team struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type ProjectCreationRequest struct {
	Name   string `json:"name"`
	TeamID int    `json:"teamID"`
}

// TODO is putting user id in query safe?
func getProjectsForUser(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("user") == "" {
		// TODO handle error
	}

	user := r.URL.Query().Get("user")
	db := createDB()
	defer db.Close()

	// get teams of user
	teams, err := db.Query("SELECT teamid FROM teampermission WHERE userid = ? ", user)

	if err != nil {
		fmt.Println("error!") // TODO print the error
		fmt.Fprintf(w, `
		{
			"status": "database went bonk"
		}`)
		return
	}
	var teamids []int
	for teams.Next() {
		var teamid = -1
		teams.Scan(&teamid)
		if teamid == -1 {
			fmt.Println("no teams! this branch is kinda dead and probably doesnt get reached")
			break
		}
		teamids = append(teamids, teamid)
	}
	fmt.Println(teamids)
	defer teams.Close()

	// get projects for all user's teams
	var projects []Project
	for i := range teamids {
		projectdto, err := db.Query("SELECT pid, title, name FROM project INNER JOIN team WHERE team.teamid = ?", i)
		if err != nil {
			fmt.Println("error!") // TODO print error
			fmt.Fprintf(w, `
			{
				"status": "database went bonk"
			}`)
			return
		}
		for projectdto.Next() {
			var p Project
			projectdto.Scan(&p.Id, &p.Name, &p.Team)

			projects = append(projects, p)
		}
	}

	// get teams where user is manager
	var managers []Team
	teamdto, err := db.Query("SELECT team.teamid, name FROM team INNER JOIN teampermission as tp WHERE tp.userid = ? AND tp.level >= 1", user)

	for teamdto.Next() {
		var t Team
		teamdto.Scan(&t.Id, &t.Name)
		managers = append(managers, t)
	}
	b, err := json.Marshal(projects)
	bt, err := json.Marshal(managers)
	fmt.Fprintf(w, `
	{
		"user_id": "%s",
		"projects": %s,
		"managed_teams": %s
	}
	`, user, string(b), string(bt))
}

func createProject(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	db := createDB()
	defer db.Close()

	var request ProjectCreationRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	_ = err // TODO

	// check permission level in team
	fmt.Println(request)
	permission, err := db.Query("SELECT level FROM teampermission WHERE userid = ? and teamid = ?", claims.Subject, request.TeamID)
	var level = -1
	for permission.Next() {
		permission.Scan(&level)
	}
	if level < 1 {
		fmt.Fprintf(w, `
		{
			"status": "no permission"
		}`)
		return
	}

	// check for unique name
	var count = 0
	hehe, err := db.Query("SELECT COUNT(*) FROM project WHERE teamid = ? and title=?", request.TeamID, request.Name)
	for hehe.Next() {
		hehe.Scan(&count)
	}
	if count > 0 {
		fmt.Fprintf(w, `
		{
			"status": "project name exists"
		}`)
		return
	}

	// TODO do we need to do something w/ base commit?
	db.Exec("INSERT INTO project(title, teamid) VALUES (?, ?)", request.Name, request.TeamID)
	_ = claims // temp

	fmt.Fprintf(w,
		`
	{
		"status": "success"
	}
	`)
}
