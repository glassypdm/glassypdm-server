// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package sqlcgen

import (
	"database/sql"
)

type Block struct {
	Hash  string `json:"hash"`
	S3key string `json:"s3key"`
	Size  int64  `json:"size"`
}

type Commit struct {
	Commitid  int64         `json:"commitid"`
	Projectid int64         `json:"projectid"`
	Userid    string        `json:"userid"`
	Comment   string        `json:"comment"`
	Numfiles  int64         `json:"numfiles"`
	Cno       sql.NullInt64 `json:"cno"`
	Timestamp int64         `json:"timestamp"`
}

type File struct {
	Projectid   int64          `json:"projectid"`
	Path        string         `json:"path"`
	Locked      int64          `json:"locked"`
	Lockownerid sql.NullString `json:"lockownerid"`
}

type Filerevision struct {
	Frid       int64         `json:"frid"`
	Projectid  int64         `json:"projectid"`
	Path       string        `json:"path"`
	Commitid   int64         `json:"commitid"`
	Hash       string        `json:"hash"`
	Frno       sql.NullInt64 `json:"frno"`
	Changetype int64         `json:"changetype"`
}

type Project struct {
	Projectid int64  `json:"projectid"`
	Title     string `json:"title"`
	Teamid    int64  `json:"teamid"`
}

type Projectpermission struct {
	Userid    string `json:"userid"`
	Projectid int64  `json:"projectid"`
	Level     int64  `json:"level"`
}

type Team struct {
	Teamid int64         `json:"teamid"`
	Name   string        `json:"name"`
	Planid sql.NullInt64 `json:"planid"`
}

type Teampermission struct {
	Userid string `json:"userid"`
	Teamid int64  `json:"teamid"`
	Level  int64  `json:"level"`
}
