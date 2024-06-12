package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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

/*
*
body:
- projectid, teamid
- proposed commit number
- commit msg
- files: [
{
filepath
size
number of chunks
list of hashes
}
]
*/
type File struct {
	Path      string   `json:"path"`
	Size      int      `json:"size"`
	NumChunks int      `json:"num_chunks"`
	Hashes    []string `json:"hashes"`
}

type CommitRequest struct {
	ProjectId         int    `json:"projectId"`
	TeamId            int    `json:"teamId"`
	Message           string `json:"message"`
	TentativeCommitId int    `json:"tentativeCommitId"`
	Files             []File `json:"files"`
}

type ProjectCreationRequest struct {
	Name   string `json:"name"`
	TeamID int    `json:"teamId"`
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
		fmt.Fprintf(w, `{ "status": "database went bonk" }`)
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

	if len(teamids) == 0 {
		fmt.Println("no teams found")
		fmt.Fprintf(w, `
		{
			"user_id": "%s",
			"projects": %s,
			"managed_teams": %s
		}
		`, user, "[]", "[]")
		return
	}

	// get projects for all user's teams
	var projects []Project
	for i := range teamids {
		projectdto, err := db.Query("SELECT pid, title, name FROM project INNER JOIN team WHERE team.teamid = ? AND project.teamid = ?", i, i)
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
	teamdto, err := db.Query("SELECT DISTINCT team.teamid, name FROM team INNER JOIN teampermission as tp WHERE tp.userid = ? AND tp.level >= 2", user)

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

	fmt.Fprintf(w,
		`
	{
		"status": "success"
	}
	`)
}

func getProjectInfo(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("pid") == "" {
		// TODO handle error
	}

	pid, err := strconv.Atoi(r.URL.Query().Get("pid"))
	_ = err

	db := createDB()
	defer db.Close()

	rows, err := db.Query("SELECT title FROM project WHERE pid = ?", pid)

	var projectname = ""
	for rows.Next() {
		rows.Scan(&projectname)
	}

	fmt.Fprintf(w, `
	{
		"title": "%s"
	}
	`, projectname)
}

// 0 (not found and not in team): no permission at all
// 1 (not found but in team): read only
// 2 (found): write access
// 3 (manager): manager, can add write access
// 4 (owner): can set managers
func getProjectPermissionByID(userId string, projectId int, teamId int) int {
	teamPermission := checkPermissionByID(teamId, userId)
	// not in team: < 1
	if teamPermission < 1 {
		return 0
	}

	db := createDB()
	defer db.Close()

	queryresult := db.QueryRow("SELECT level FROM projectpermission WHERE userid = ? AND projectid = ?", userId, projectId)
	var level int
	err := queryresult.Scan(&level)
	if err == sql.ErrNoRows {
		return 1 // read only
	} else if err != nil {
		return 0 // general error/no permission
	}

	return level
}

/*
*
body:
- projectid, teamid
- proposed commit number
- commit msg
- files: [
{
filepath
size
number of chunks
list of hashes
}
]
*/
func commit(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	userId := claims.Subject
	var request CommitRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		fmt.Fprintf(w, `{ "response": "bad json" }`)
	}

	// check permission
	projectPermission := getProjectPermissionByID(userId, request.ProjectId, request.TeamId)
	if projectPermission < 2 {
		fmt.Fprintf(w, `{ "response": "no permission" }`)
		return
	}

	// TODO
	// iterate through hashes to see if we have it in S3 (can see thru block table)
	// if we need hashes, return nb
	// otherwise, commit
	var hashesMissing []string
	/*
		for _, file := range request.Files {

		}
	*/
	if len(hashesMissing) > 0 {
		// respond with nb
		return
	}

	// TODO
	// no hashes missing, so commit
	// make an entry in the commit, file, and filerevision tables
}

// given a project id, returns the newest commit id used
func getLatestCommit(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	userId := claims.Subject
	project := r.URL.Query().Get("projectId")
	pid, err := strconv.Atoi(project)
	if err != nil {
		fmt.Fprintf(w, `{ "response": "incorrect format" }`)
		return
	}

	db := createDB()
	defer db.Close()

	// check user permissions
	// needs at least read permission
	rows, err := db.Query("SELECT COUNT(*) FROM teampermission WHERE userid = ?", userId)
	if err != nil {
		fmt.Fprintf(w, `{ "response": "database issue" }`)
		return
	}
	var count int
	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			fmt.Fprintf(w, `{ "response": "database issue" }`)
			return
		}
	}
	if count < 1 {
		fmt.Fprintf(w, `{ "response": "invalid permission" }`)
	}

	// get latest commit for pid
	rows, err = db.Query("SELECT MAX(cid) FROM 'commit' WHERE projectid = ?", pid)
	if err != nil {
		fmt.Fprintf(w, `{ "response": "database issue" }`)
		return
	}
	var commit int
	for rows.Next() {
		rows.Scan(&commit)
	}
	fmt.Fprintf(w, `
	{
		"response": "valid",
		"newestCommit": %d
	}`, commit)
}
