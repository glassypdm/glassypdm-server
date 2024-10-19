package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
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
	log.Info("creating commit..")
	userId := claims.Subject
	var request CommitRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		PrintError(w, "bad json")
		return
	}

	// check permission
	projectPermission := getProjectPermissionByID(userId, request.ProjectId)
	if projectPermission < 2 {
		PrintError(w, "no permission")
		return
	}
	start := time.Now()

	tx, err := db_pool.Begin(ctx)
	if err != nil {
		log.Error("couldn't create transaction", "error", err)
		PrintError(w, "db error")
		return
	}
	defer tx.Rollback(ctx)
	qtx := queries.WithTx(tx)
	// make commit, get new commitid
	cid, err := qtx.InsertCommit(ctx, sqlcgen.InsertCommitParams{
		Projectid: int32(request.ProjectId),
		Userid:    userId,
		Comment:   request.Message,
		Numfiles:  int32(len(request.Files))})
	if err != nil {
		log.Error("db couldn't create commit", "db err", err)
		PrintError(w, "db error")
		return
	}

	var hashesMissing []string
	//for _, file := range request.Files {
	for i := 0; i < len(request.Files); i += 2 {

		// FIXME these dont add numchunks
		if i+1 >= len(request.Files) {
			err = qtx.InsertFileRevision(ctx, sqlcgen.InsertFileRevisionParams{
				Projectid:  int32(request.ProjectId),
				Path:       request.Files[i].Path,
				Commitid:   cid,
				Filehash:   request.Files[i].Hash,
				Changetype: int32(request.Files[i].ChangeType)})
		} else {
			err = qtx.InsertTwoFileRevisions(ctx, sqlcgen.InsertTwoFileRevisionsParams{
				Projectid:    int32(request.ProjectId),
				Path:         request.Files[i].Path,
				Commitid:     cid,
				Filehash:     request.Files[i].Hash,
				Changetype:   int32(request.Files[i].ChangeType),
				Projectid_2:  int32(request.ProjectId),
				Path_2:       request.Files[i+1].Path,
				Commitid_2:   cid,
				Filehash_2:   request.Files[i+1].Hash,
				Changetype_2: int32(request.Files[i+1].ChangeType)})
		}

		if err != nil {
			if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") || strings.Contains(err.Error(), "foreign key mismatch") {
				// TODO error handling here isnt very ergonomic i think
				hashesMissing = append(hashesMissing, request.Files[i].Hash)
				hashesMissing = append(hashesMissing, request.Files[i+1].Hash)
				continue
			} else {
				log.Error("now-handled error inserting file revision", "db", err)
				PrintError(w, "db error")
				return
			}
		}
	}
	durationOne := time.Since(start)
	log.Info("iterating took " + durationOne.String() + " over " + fmt.Sprint(len(request.Files)) + " files")
	hashes_bytes, _ := json.Marshal(hashesMissing)
	if len(hashesMissing) > 0 {
		log.Warn("found missing hashes", "len", len(hashesMissing))
		// respond with nb
		PrintResponse(w, "nb", string(hashes_bytes))
		return
	}

	// no hashes missing, so commit the transaction
	// we should consider returning more info too
	tx.Commit(ctx)

	output := CreateCommitOutput{CommitId: int(cid)}
	output_bytes, _ := json.Marshal(output)
	durationTwo := time.Since(start)
	log.Info("transaction took " + durationTwo.String())
	PrintSuccess(w, string(output_bytes))
}

type CreateCommitOutput struct {
	CommitId int `json:"commit_id"`
}

// input: query offset=<number>
// returns:
// {
// # of commits
// commit object list
// }

type CommitDescription struct {
	CommitId     int    `json:"commit_id"`
	CommitNumber int    `json:"commit_no"`
	NumFiles     int    `json:"num_files"`
	Author       string `json:"author"`
	Comment      string `json:"comment"`
	Timestamp    int64  `json:"timestamp"`
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
		PrintError(w, "incorrect format")
		return
	}

	if r.URL.Query().Get("offset") == "" {
		PrintError(w, "incorrect format")
		return
	}
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		PrintError(w, "incorrect format")
		return
	}

	// check if user has read permission for project
	if getProjectPermissionByID(userId, pid) < 1 {
		PrintError(w, "no permission")
		return
	}

	// get commits
	CommitDto, err := queries.ListProjectCommits(ctx, sqlcgen.ListProjectCommitsParams{Projectid: int32(pid), Offset: int32(offset)})
	if err != nil {
		log.Error("db error", "sql", err.Error())
		PrintError(w, "db error")
		return
	}
	// get total number
	NumCommits, err := queries.CountProjectCommits(ctx, int32(pid))
	if err != nil {
		log.Error("db error", "sql", err.Error())
		PrintError(w, "db error")
		return
	}

	var CommitDescriptions []CommitDescription
	for _, Commit := range CommitDto {
		// get author from clerk + userid
		usr, err := user.Get(ctx, Commit.Userid)
		name := ""
		if err != nil {
			PrintError(w, "invalid user id")
			return
		}
		name = *usr.FirstName + " " + *usr.LastName

		CommitDescriptions = append(CommitDescriptions, CommitDescription{
			CommitId:     int(Commit.Commitid),
			CommitNumber: int(Commit.Cno.Int32),
			NumFiles:     int(Commit.Numfiles),
			Comment:      Commit.Comment,
			Timestamp:    Commit.Timestamp.Time.UnixNano() / 1000000000,
			Author:       name,
		})
	}

	output := CommitList{NumCommit: int(NumCommits), Commits: CommitDescriptions}
	JSONList, err := json.Marshal(output)
	if err != nil {
		PrintError(w, "json error")
		return
	}
	PrintSuccess(w, string(JSONList))
}

func GetCommitInformation(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	_ = ctx
	_ = claims

	// validate commit id
	CommitIdStr := chi.URLParam(r, "commit-id")
	CommitId, err := strconv.Atoi(CommitIdStr)
	if err != nil {
		fmt.Fprintf(w, `{ "response": "incorrect format" }`)
		PrintError(w, "incorrect format")
		return
	}

	// get commit information
	CommitInfoDto, err := queries.GetCommitInfo(ctx, int32(CommitId))
	if err != nil {
		PrintError(w, "db error")
		log.Warn("encountered db error when getting commit info", "db", err, "commit-id", CommitId)
		return
	}

	// check permission - needs read permission minimum
	if getProjectPermissionByID(claims.Subject, int(CommitInfoDto.Projectid)) < 1 {
		log.Warn("insufficient permission", "user", claims.Subject, "projectId", CommitInfoDto.Projectid)
		PrintError(w, "insufficient permission")
		return
	}

	// get file revisions
	Files, err := queries.GetFileRevisionsByCommitId(ctx, int32(CommitId))
	if err != nil {
		PrintError(w, "db error")
		log.Warn("encountered db error when getting file revisions for commit", "db", err, "commit-id", CommitId)
		return
	}

	var Output CommitInformation
	Output.FilesChanged = Files

	usr, err := user.Get(ctx, CommitInfoDto.Userid)
	name := ""
	if err != nil {
		PrintError(w, "invalid user id")
		return
	}
	name = *usr.FirstName + " " + *usr.LastName

	Output.Description = CommitDescription{
		CommitId:     CommitId,
		CommitNumber: int(CommitInfoDto.Cno.Int32),
		NumFiles:     int(CommitInfoDto.Numfiles),
		Comment:      CommitInfoDto.Comment,
		Timestamp:    CommitInfoDto.Timestamp.Time.UnixNano() / 1000000000,
		Author:       name,
	}

	OutputJson, err := json.Marshal(Output)
	if err != nil {
		PrintError(w, "json error")
		return
	}
	PrintSuccess(w, string(OutputJson))
}

type CommitInformation struct {
	Description  CommitDescription                       `json:"description"`
	FilesChanged []sqlcgen.GetFileRevisionsByCommitIdRow `json:"files"`
}
