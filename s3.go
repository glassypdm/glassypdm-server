package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/joshtenorio/glassypdm-server/sqlcgen"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"lukechampine.com/blake3"
)

func generateS3Client() (*minio.Client, error) {
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKeyID := os.Getenv("S3_ACCESSKEYID")
	secretAccessKey := os.Getenv("S3_SECRETKEY")
	useSSL := true

	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
}

/*
steps:
- check permission for uploading in general
- reads file, upload to s3
- compares user-supplied hash w/ our own hashing. if they match, we put thing in db. otherwise we delete from s3
*/
func HandleUpload(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// note: this size here is just for parsing and not the actual size limit of the file
	// TODO is this note correct?
	if err := r.ParseMultipartForm(400 * (1 << 20)); err != nil { // 400 * (1 << 20) is 400 MB
		WriteError(w, "multipart form parsing failed")
		return
	}

	UserId := r.FormValue("user_id")
	if UserId == "" {
		WriteError(w, "form format incorrect")
		return
	}

	// ensure user can upload to at least one project/team
	if !canUserUpload(UserId) {
		WriteError(w, "no upload permission")
		return
	}

	FileHash := r.FormValue("file_hash")
	if FileHash == "" {
		WriteError(w, "form format incorrect")
		return
	}

	ChunkIndex := r.FormValue("chunk_index")
	if ChunkIndex == "" {
		WriteError(w, "form format incorrect")
		return
	}
	NumChunks := r.FormValue("num_chunks")
	if NumChunks == "" {
		WriteError(w, "form format incorrect")
		return
	}

	file, header, err := r.FormFile("chunk")
	if err != nil {
		WriteError(w, "cannot read file")
		return
	}
	size := header.Size

	cidx, err1 := strconv.ParseInt(ChunkIndex, 10, 32)
	numchunks, err2 := strconv.ParseInt(NumChunks, 10, 32)
	if err1 != nil || err2 != nil {
		WriteError(w, "form format incorrect")
		return
	}

	hashUser := r.FormValue("block_hash")
	if hashUser == "" {
		WriteError(w, "form format incorrect")
		return
	}

	// set position back to start.
	// TODO do we need this?
	if _, err := file.Seek(0, 0); err != nil {
		WriteError(w, "error reading file")
		log.Error("couldn't read file", "err", err.Error())
		return
	}

	s3, err := generateS3Client()
	if err != nil {
		log.Error("couldn't connect to s3", "err", err.Error())
		WriteError(w, "issue connecting to s3")
		return
	}
	hasher := blake3.New(32, nil)
	tee := io.TeeReader(file, hasher)

	// check if object exists in S3 already
	err = queries.InsertHash(ctx,
		sqlcgen.InsertHashParams{Blockhash: hashUser, S3key: hashUser, Blocksize: int32(size)})
	if err != nil {
		log.Error("couldn't insert hash", "db", err.Error())
		var e *pgconn.PgError
		if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
			log.Warn("found duplicate hash", "hash", hashUser)

			// insert the chunk because we need to anyways
			err = queries.InsertChunk(ctx, sqlcgen.InsertChunkParams{
				Chunkindex: int32(cidx),
				Numchunks:  int32(numchunks),
				Filehash:   FileHash,
				Blockhash:  hashUser,
				Blocksize:  int32(size),
			})
			if err != nil {
				var e *pgconn.PgError
				if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
					// chunk exists already
				} else {
					log.Error("couldn't insert chunk", "db", err.Error())
					WriteError(w, "db error")
					return
				}
			}

			WriteDefaultSuccess(w, "duplicate found")
			return
		}
	}

	// insert object into S3
	_, err = s3.PutObject(
		ctx,
		os.Getenv("S3_BUCKETNAME"),
		hashUser,
		tee,
		size,
		minio.PutObjectOptions{ContentType: "application/octet-stream"})

	if err != nil {
		log.Error("couldn't connect to s3", "s3", err.Error())
		WriteError(w, "issue connecting to s3")
		return
	}

	// confirm hash matches what the user supplies us
	// if hash does not match, remove from bucket and db
	hashCalc := hasher.Sum(nil)
	if hashUser != hex.EncodeToString(hashCalc) {
		log.Error("hash doesn't match", "user", hashUser, "calculated", hashCalc)
		WriteError(w, "hash doesn't match")
		s3.RemoveObject(
			ctx,
			os.Getenv("S3_BUCKETNAME"),
			hashUser,
			minio.RemoveObjectOptions{})

		queries.RemoveHash(ctx, hashUser)
		return
	}

	err = queries.InsertChunk(ctx, sqlcgen.InsertChunkParams{
		Chunkindex: int32(cidx),
		Numchunks:  int32(numchunks),
		Filehash:   FileHash,
		Blockhash:  hashUser,
		Blocksize:  int32(size),
	})
	if err != nil {
		var e *pgconn.PgError
		if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
			log.Warn("duplicate found") // TODO downgrade to ifno
		} else {
			log.Error("couldn't insert chunk", "sql", err.Error())
			WriteError(w, "db error")
			return
		}

	}
	WriteDefaultSuccess(w, "upload successful")
}

type FileChunk struct {
	Url       string `json:"s3_url"`
	BlockHash string `json:"block_hash"`
	FileHash  string `json:"file_hash"`
	Index     int    `json:"chunk_index"`
}
type DownloadOutput struct {
	FileHash string      `json:"file_hash"`
	CommitId int         `json:"commit_id"`
	FilePath string      `json:"file_path"`
	Chunks   []FileChunk `json:"file_chunks"`
}

type DownloadRequest struct {
	ProjectId int    `json:"project_id"`
	Path      string `json:"path"`
	CommitId  int    `json:"commit_id"`
	UserId    string `json:"user_id"`
}

func GetS3Download(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var request DownloadRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		WriteError(w, "bad json")
		return
	}

	// check permission level
	if getProjectPermissionByID(request.UserId, request.ProjectId) < 1 {
		WriteError(w, "no permission")
		return
	}

	s3, err := generateS3Client()
	if err != nil {
		log.Error("couldn't connect to s3", "s3", err.Error())
		WriteError(w, "issue connecting to s3")
		return
	}

	// get filehash from filepath+projectid
	filehash, err := queries.GetFileHash(ctx,
		sqlcgen.GetFileHashParams{
			Projectid: int32(request.ProjectId),
			Path:      request.Path,
			Commitid:  int32(request.CommitId),
		})
	if err != nil {
		log.Error("couldn't get filehash", "projectID", request.ProjectId, "filepath", request.Path, "db err", err.Error())
		WriteError(w, "db error")
		return
	}

	// ping s3 for a presigned url
	chunksDto, err := queries.GetFileChunks(ctx, filehash)
	if err != nil {
		log.Error("coudln't get file chunks", "filehash", filehash, "db err", err.Error())
		WriteError(w, "db error")
		return
	}

	var chunks []FileChunk
	for _, chunk := range chunksDto {
		// ping s3 for a presigned url
		reqParams := make(url.Values)
		url, err := s3.PresignedGetObject(ctx, os.Getenv("S3_BUCKETNAME"), chunk.Blockhash, time.Second*60*60*48, reqParams)
		if err != nil {
			log.Error("couldn't get presigned GET link", "s3", err.Error())
			WriteError(w, "s3 error")
			return
		}
		chunks = append(chunks,
			FileChunk{
				Url:       url.String(),
				BlockHash: chunk.Blockhash,
				Index:     int(chunk.Chunkindex),
				FileHash:  filehash})
	}

	// return result
	output := DownloadOutput{
		FileHash: filehash,
		CommitId: request.CommitId,
		FilePath: request.Path,
		Chunks:   chunks,
	}

	output_str, _ := json.Marshal(output)
	WriteSuccess(w, string(output_str))
}
