-- name: InsertChunk :exec
INSERT INTO chunk(chunkindex, numchunks, filehash, blockhash, blocksize)
VALUES ($1, $2, $3, $4, $5);

-- name: InsertHash :exec
INSERT INTO block(blockhash, s3key, blocksize)
VALUES ($1, $2, $3);

-- name: RemoveHash :exec
DELETE FROM block WHERE blockhash = $1;

-- name: GetS3Key :one
SELECT s3key FROM block
WHERE blockhash = $1 LIMIT 1;

-- name: GetFileChunks :many
SELECT blockhash, chunkindex FROM chunk
WHERE filehash = $1 ORDER BY chunkindex ASC;
