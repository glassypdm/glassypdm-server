-- name: GetProject :one
SELECT * FROM project WHERE pid = ? LIMIT 1;