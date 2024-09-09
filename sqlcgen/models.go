// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package sqlcgen

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type Block struct {
	Blockhash string `json:"blockhash"`
	S3key     string `json:"s3key"`
	Blocksize int32  `json:"blocksize"`
}

type Chunk struct {
	Chunkindex int32  `json:"chunkindex"`
	Numchunks  int32  `json:"numchunks"`
	Filehash   string `json:"filehash"`
	Blockhash  string `json:"blockhash"`
	Blocksize  int32  `json:"blocksize"`
}

type Commit struct {
	Commitid  int32       `json:"commitid"`
	Projectid int32       `json:"projectid"`
	Userid    string      `json:"userid"`
	Comment   string      `json:"comment"`
	Numfiles  int32       `json:"numfiles"`
	Cno       pgtype.Int4 `json:"cno"`
	Timestamp int32       `json:"timestamp"`
}

type File struct {
	Projectid   int32       `json:"projectid"`
	Path        string      `json:"path"`
	Locked      int32       `json:"locked"`
	Lockownerid pgtype.Text `json:"lockownerid"`
}

type Filerevision struct {
	Frid       int32       `json:"frid"`
	Projectid  int32       `json:"projectid"`
	Path       string      `json:"path"`
	Commitid   int32       `json:"commitid"`
	Filehash   string      `json:"filehash"`
	Changetype int32       `json:"changetype"`
	Numchunks  int32       `json:"numchunks"`
	Frno       pgtype.Int4 `json:"frno"`
}

type Permissiongroup struct {
	Pgroupid int32  `json:"pgroupid"`
	Teamid   int32  `json:"teamid"`
	Name     string `json:"name"`
}

type Pgmapping struct {
	Pgroupid  int32 `json:"pgroupid"`
	Projectid int32 `json:"projectid"`
	Level     int32 `json:"level"`
}

type Pgmembership struct {
	Pgroupid int32  `json:"pgroupid"`
	Userid   string `json:"userid"`
}

type Project struct {
	Projectid int32  `json:"projectid"`
	Title     string `json:"title"`
	Teamid    int32  `json:"teamid"`
}

type Team struct {
	Teamid int32       `json:"teamid"`
	Name   string      `json:"name"`
	Planid pgtype.Int4 `json:"planid"`
}

type Teampermission struct {
	Userid string `json:"userid"`
	Teamid int32  `json:"teamid"`
	Level  int32  `json:"level"`
}
