package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	fmt.Println("creating commit..")
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
	start := time.Now()

	tx, qtx := useTxQueries()
	defer tx.Rollback()
	// make commit, get new commitid
	cid, err := qtx.InsertCommit(ctx, sqlcgen.InsertCommitParams{
		Projectid: int64(request.ProjectId),
		Userid:    userId,
		Comment:   request.Message,
		Numfiles:  int64(len(request.Files))})
	if err != nil {
		fmt.Println("error creating commit")
		fmt.Fprintf(w, `
		{
		"status": "error"
		}`)
		return
	}

	var hashesMissing []string
	//for _, file := range request.Files {
	for i := 0; i < len(request.Files); i += 2 {

		if i+1 >= len(request.Files) {
			err = qtx.InsertFileRevision(ctx, sqlcgen.InsertFileRevisionParams{
				Projectid:  int64(request.ProjectId),
				Path:       request.Files[i].Path,
				Commitid:   cid,
				Filehash:   request.Files[i].Hash,
				Changetype: int64(request.Files[i].ChangeType)})
		} else {
			err = qtx.InsertTwoFileRevisions(ctx, sqlcgen.InsertTwoFileRevisionsParams{
				Projectid:    int64(request.ProjectId),
				Path:         request.Files[i].Path,
				Commitid:     cid,
				Filehash:     request.Files[i].Hash,
				Changetype:   int64(request.Files[i].ChangeType),
				Projectid_2:  int64(request.ProjectId),
				Path_2:       request.Files[i+1].Path,
				Commitid_2:   cid,
				Filehash_2:   request.Files[i+1].Hash,
				Changetype_2: int64(request.Files[i+1].ChangeType)})
		}

		if err != nil {
			if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
				// TODO error handling here isnt very ergonomic i think
				hashesMissing = append(hashesMissing, request.Files[i].Hash)
				hashesMissing = append(hashesMissing, request.Files[i+1].Hash)
				continue
			} else {
				fmt.Printf("error %v\n", err)
			}
		}
	}
	durationOne := time.Since(start)
	fmt.Println("iterating took " + durationOne.String() + " over " + fmt.Sprint(len(request.Files)) + " files")
	asdfjkl, err := json.Marshal(hashesMissing)
	if len(hashesMissing) > 0 {
		// respond with nb
		fmt.Fprintf(w, `
			{
			"status": "nb",
			"hashes": %v
			}`, asdfjkl)
		return
	}

	// no hashes missing, so commit the transaction
	// we should consider returning more info too
	tx.Commit()

	durationTwo := time.Since(start)
	fmt.Println("transaction took " + durationTwo.String())
	fmt.Fprintf(w, `{
	"status": "success",
	"commitid": %v
	}`, cid)
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
