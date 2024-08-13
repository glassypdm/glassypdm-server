package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/go-chi/chi/v5"
	"github.com/joshtenorio/glassypdm-server/sqlcgen"
)

/*
*
body:
- projectid, teamid
- CreateCommit msg
- files: [
{
filepath
hash
changetype
}
]
*/
func CreateCommit(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
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
		return
	}

	// check permission
	projectPermission := getProjectPermissionByID(userId, request.ProjectId)
	if projectPermission < 2 {
		fmt.Fprintf(w, `{ "response": "no permission" }`)
		return
	}

	tx, qtx := useTxQueries()
	defer tx.Rollback()
	// make commit, get new commitid
	cid, err := qtx.InsertCommit(ctx, sqlcgen.InsertCommitParams{
		Projectid: int64(request.ProjectId),
		Userid:    userId,
		Comment:   request.Message,
		Numfiles:  int64(len(request.Files))})

	// FIXME if we create a new file entry
	// we don't see it when we have a filerevision
	var hashesMissing []string
	for _, file := range request.Files {
		_ = file
		// insert into file
		err = qtx.InsertFile(ctx, sqlcgen.InsertFileParams{Projectid: int64(request.ProjectId), Path: file.Path})
		if err != nil {
			// TODO do we need to handle anything here?
			fmt.Println("uwuwuwu")

			fmt.Printf("err: %v\n", err)
		}

		// add filerevision
		// error if we fail a unique thing (hopefully)
		err = qtx.InsertFileRevision(ctx, sqlcgen.InsertFileRevisionParams{
			Projectid:  int64(request.ProjectId),
			Path:       file.Path,
			Commitid:   cid,
			Hash:       file.Hash,
			Changetype: int64(file.ChangeType)})
		if err != nil {
			// TODO confirm error
			fmt.Printf("error %v\n", err)
			hashesMissing = append(hashesMissing, file.Hash)
			continue
		}

	}
	if len(hashesMissing) > 0 {
		// respond with nb
		fmt.Fprintf(w, `
			{
			"status": "nb",
			"hashes": %v
			}`, hashesMissing)
		return
	}

	// no hashes missing, so commit the transaction
	// we should consider returning more info too
	tx.Commit()
	fmt.Fprintf(w, `{
	"status": "success",
	"commitid": %v
	}`, cid)
}

// given a project id, returns the newest commit id used
func GetLatestCommit(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	userId := claims.Subject
	project := r.URL.Query().Get("project-id")
	pid, err := strconv.Atoi(project)
	if err != nil {
		fmt.Fprintf(w, `{ "response": "incorrect format" }`)
		return
	}

	db := CreateDB()
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

// input: query offset=<number>
// returns:
// {
// # of commits
// commit object list
// }

type CommitDescription struct {
	CommitId     int    `json:"commit_id"`
	CommitNumber int    `json:"commit_number"`
	NumFiles     int    `json:"num_files"`
	Author       string `json:"author"`
	Comment      string `json:"comment"`
	Timestamp    int    `json:"timestamp"`
}

type CommitList struct {
	NumCommit int                 `json:"num_commits"`
	Commits   []CommitDescription `json:"commits"`
}

func GetCommits(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	userId := claims.Subject
	project := chi.URLParam(r, "project-id")
	pid, err := strconv.Atoi(project)
	if err != nil {
		fmt.Fprintf(w, `{ "response": "incorrect format" }`)
		return
	}

	if r.URL.Query().Get("offset") == "" {
		fmt.Fprintf(w, `{ "response": "incorrect format" }`)
		return
	}
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))

	// check if user has read permission for project
	if getProjectPermissionByID(userId, pid) < 1 {
		fmt.Fprintf(w, `{ "response": "error", "error": "no permission" }`)
		return
	}

	query := UseQueries()

	// get commits
	CommitDto, err := query.ListProjectCommits(ctx, sqlcgen.ListProjectCommitsParams{Projectid: int64(pid), Offset: int64(offset)})
	if err != nil {
		fmt.Println(err)
		PrintError(w, "db error")
		return
	}
	// get total number
	NumCommits, err := query.CountProjectCommits(ctx, int64(pid))
	if err != nil {
		fmt.Println(err)
		PrintError(w, "db error")
		return
	}

	var CommitDescriptions []CommitDescription
	for _, Commit := range CommitDto {
		// get author from clerk + userid
		usr, err := user.Get(ctx, userId)
		name := ""
		if err != nil {
			PrintError(w, "invalid user id")
		}
		name = *usr.FirstName + " " + *usr.LastName

		// TODO verify commit number
		CommitDescriptions = append(CommitDescriptions, CommitDescription{
			CommitId:     int(Commit.Commitid),
			CommitNumber: int(Commit.Cno.Int64),
			NumFiles:     int(Commit.Numfiles),
			Comment:      Commit.Comment,
			Timestamp:    int(Commit.Timestamp),
			Author:       name,
		})
	}

	output := CommitList{NumCommit: int(NumCommits), Commits: CommitDescriptions}
	JSONList, err := json.Marshal(output)
	if err != nil {
		PrintError(w, "json error")
	}
	PrintSuccess(w, string(JSONList))
}