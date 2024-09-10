package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/clerk/clerk-sdk-go/v2/user"
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joshtenorio/glassypdm-server/sqlcgen"
)

func IsServerOpen() bool {
	if os.Getenv("OPEN_TEAMS") == "1" {
		return true
	} else {
		return false
	}
}
func getVersion(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Version string `json:"version"`
	}{}
	data.Version = os.Getenv("CLIENT_VERSION")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func getConfig(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Key  string `json:"clerk_publickey"`
		Name string `json:"name"`
	}{}
	data.Key = os.Getenv("CLERK_PUBLICKEY")
	data.Name = os.Getenv("SERVER_NAME")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

var queries sqlcgen.Queries
var db_pool *pgxpool.Pool

//go:embed schema.sql
var ddl string

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

	// check team permission
	// TODO: ensure that a team has at least one project for this to be a valid check
	teampermissions, err := queries.FindTeamPermissions(ctx, userId)
	if err != nil {
		return false
	}
	for _, level := range teampermissions {
		if level >= 2 {
			return true
		}
	}

	// check permission groups
	groups, err := queries.FindUserInPermissionGroup(ctx, userId)
	if err != nil {
		return false
	}
	if len(groups) > 0 {
		return true
	}
	return false
}

type User struct {
	UserId  string `json:"user_id"`
	Name    string `json:"name"`
	EmailId string `json:"email_id"`
}

func GetUserByID(userId string) (User, bool) {
	ctx := context.Background()
	var output User
	usr, err := user.Get(ctx, userId)
	if err != nil {
		log.Error("couldn't find user", "user", userId, "error", err.Error())
		return output, false
	}

	output.UserId = userId
	output.Name = *usr.FirstName + " " + *usr.LastName
	output.EmailId = *usr.PrimaryEmailAddressID

	return output, true
}
