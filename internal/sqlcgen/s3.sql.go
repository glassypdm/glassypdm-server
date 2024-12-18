// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: s3.sql

package sqlcgen

import (
	"context"
)

const getFileChunks = `-- name: GetFileChunks :many
SELECT blockhash, chunkindex FROM chunk
WHERE filehash = $1 ORDER BY chunkindex ASC
`

type GetFileChunksRow struct {
	Blockhash  string `json:"blockhash"`
	Chunkindex int32  `json:"chunkindex"`
}

func (q *Queries) GetFileChunks(ctx context.Context, filehash string) ([]GetFileChunksRow, error) {
	rows, err := q.db.Query(ctx, getFileChunks, filehash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetFileChunksRow
	for rows.Next() {
		var i GetFileChunksRow
		if err := rows.Scan(&i.Blockhash, &i.Chunkindex); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getS3Key = `-- name: GetS3Key :one
SELECT s3key FROM block
WHERE blockhash = $1 LIMIT 1
`

func (q *Queries) GetS3Key(ctx context.Context, blockhash string) (string, error) {
	row := q.db.QueryRow(ctx, getS3Key, blockhash)
	var s3key string
	err := row.Scan(&s3key)
	return s3key, err
}

const insertChunk = `-- name: InsertChunk :exec
INSERT INTO chunk(chunkindex, numchunks, filehash, blockhash, blocksize)
VALUES ($1, $2, $3, $4, $5)
`

type InsertChunkParams struct {
	Chunkindex int32  `json:"chunkindex"`
	Numchunks  int32  `json:"numchunks"`
	Filehash   string `json:"filehash"`
	Blockhash  string `json:"blockhash"`
	Blocksize  int32  `json:"blocksize"`
}

func (q *Queries) InsertChunk(ctx context.Context, arg InsertChunkParams) error {
	_, err := q.db.Exec(ctx, insertChunk,
		arg.Chunkindex,
		arg.Numchunks,
		arg.Filehash,
		arg.Blockhash,
		arg.Blocksize,
	)
	return err
}

const insertHash = `-- name: InsertHash :exec
INSERT INTO block(blockhash, s3key, blocksize)
VALUES ($1, $2, $3)
`

type InsertHashParams struct {
	Blockhash string `json:"blockhash"`
	S3key     string `json:"s3key"`
	Blocksize int32  `json:"blocksize"`
}

func (q *Queries) InsertHash(ctx context.Context, arg InsertHashParams) error {
	_, err := q.db.Exec(ctx, insertHash, arg.Blockhash, arg.S3key, arg.Blocksize)
	return err
}

const removeHash = `-- name: RemoveHash :exec
DELETE FROM block WHERE blockhash = $1
`

func (q *Queries) RemoveHash(ctx context.Context, blockhash string) error {
	_, err := q.db.Exec(ctx, removeHash, blockhash)
	return err
}