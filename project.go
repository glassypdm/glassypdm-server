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
}

func getProjectsForUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	db := createDB()
	defer db.Close()

	// get teams of user
	teams, err := db.Query("SELECT teamid FROM teampermission WHERE userid = ? ", claims.Subject)

	if err != nil {
		fmt.Println("error!") // TODO print error
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

	// get projects
	var projects []Project
	for i := range teamids {
		projectdto, err := db.Query("SELECT pid, title FROM project WHERE teamid = ?", i)
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
			projectdto.Scan(&p.Id, &p.Name)

			projects = append(projects, p)
		}
	}

	b, err := json.Marshal(projects)
	fmt.Println(projects)
	fmt.Println(string(b))
	fmt.Fprintf(w, `
	{
		"user_id": "%s",
		"projects": %s
	}
	`, claims.Subject, string(b))
}
