package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/clerk/clerk-sdk-go/v2"
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

func HandleUpload(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	// note: this size here is just for parsing and not the actual size limit of file
	if err := r.ParseMultipartForm(900 * (1 << 20)); err != nil { // 900 * (1 << 20) is 900 MB
		fmt.Fprintf(w, `{ "status": "multipart form parsing failed" }`)
		return
	}

	// ensure user can upload to at least one project/team
	if !canUserUpload(claims.Subject) {
		fmt.Fprintf(w, `{ "status": "no upload permissions" }`)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		fmt.Fprintf(w, `{ "status": "couldn't read file" }`)
		return
	}
	size := header.Size

	hashUser := r.FormValue("hash")
	if hashUser == "" {
		fmt.Fprintf(w, `{ "status": "form format incorrect" }`)
		return
	}

	// set position back to start.
	// TODO delete?
	if _, err := file.Seek(0, 0); err != nil {
		fmt.Fprintf(w, `{ "status": "issue reading" }`)
		fmt.Println(err)
		return
	}

	queries := UseQueries()

	s3, err := generateS3Client()
	if err != nil {
		fmt.Println(err)
		fmt.Fprintf(w, `{ "status": "issue connecting to s3" }`)
		return
	}
	hasher := blake3.New(32, nil)
	tee := io.TeeReader(file, hasher)

	// check if object exists in S3 already
	_, err = queries.FindHash(ctx, hashUser)
	if err != nil {
		fmt.Println(err)
		//fmt.Fprintf(w, `{ "status": "db error" }`)
	} else {
		fmt.Fprintf(w, `{ "status": "hash exists already" }`)
		return
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
		fmt.Println(err)
		fmt.Fprintf(w, `{ "status": "issue connecting to s3" }`)
		return
	}

	// confirm hash matches what the user supplies us
	hashCalc := hasher.Sum(nil)
	if hashUser != hex.EncodeToString(hashCalc) {
		fmt.Println(hashUser)
		fmt.Println(hex.EncodeToString(hashCalc))
		fmt.Fprintf(w, `{ "status": "hash doesn't match" }`)
		fmt.Println("bruh")

		// remove object from bucket
		s3.RemoveObject(
			ctx,
			os.Getenv("S3_BUCKETNAME"),
			hashUser,
			minio.RemoveObjectOptions{})

		return
	} else {
		fmt.Println("hash ok")
		// hash matches; so insert entry into database
		err = queries.InsertHash(ctx, sqlcgen.InsertHashParams{Hash: hashUser, S3key: hashUser, Size: size})
		if err != nil {
			fmt.Fprintf(w, `{ "status": "db error" }`)
		}
	}
	fmt.Fprintf(w, `{ "status": "success" }`)

	// TODO not secure!!!
	// we shouldn't let the user control the key
	// ideas to fix:
	// - hash the file here, compare results
	// - use timestamp as s3key
	// - compute a uuid somehow
	s3.PutObject(context.Background(), os.Getenv("S3_BUCKETNAME"), hash, file, file.(Sizer).Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})

	w.Write([]byte("hehez"))
}
