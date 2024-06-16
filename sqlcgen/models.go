// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package sqlcgen

import (
	"database/sql"
	"time"
)

type Block struct {
	Hash  string
	S3key string
}

type Chunk struct {
	Cid         int64
	Frid        int64
	Chunknumber int64
	Hash        string
}

type Commit struct {
	Cid       int64
	Projectid int64
	Userid    string
	Comment   string
	Numfiles  int64
	Cno       sql.NullInt64
	Timestamp time.Time
}

type File struct {
	Fid  int64
	Pid  int64
	Path string
}

type Filerevision struct {
	Frid       int64
	Fid        int64
	Commitid   int64
	Numchunks  int64
	Frno       sql.NullInt64
	Changetype int64
}

type Project struct {
	Pid    int64
	Title  string
	Teamid int64
}

type Projectpermission struct {
	Userid    string
	Projectid int64
	Level     int64
}

type Team struct {
	Teamid int64
	Name   string
	Planid sql.NullInt64
}

type Teampermission struct {
	Userid string
	Teamid int64
	Level  int64
}
