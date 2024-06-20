package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Sizer interface {
	Size() int64
}

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

// multipart
// chunk + hash
func HandleUpload(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	// TODO config file size from env
	// and send it in /config
	if err := r.ParseMultipartForm(900 * (1 << 20)); err != nil { // 900 * (1 << 20) is 900 MB.
		w.Write([]byte(`{"status": "file size too large"}`))
		return
	}

	// ensure user can upload to at least one project/team
	if !canUserUpload(claims.Subject) {
		return // TODO error message
	}

	file, _, err := r.FormFile("file")
	// Create a buffer to store the header of the file in

	// set position back to start.
	if _, err := file.Seek(0, 0); err != nil {
		fmt.Println(err)
		return
	}

	s3, err := generateS3Client()
	if err != nil {
		fmt.Println(err)
		w.Write([]byte("issue w/ s3 client"))
		return
	}

	hash := r.FormValue("hash")
	if hash == "" {
		// TODO incomplete form
		return
	}

	s3.PutObject(context.Background(), os.Getenv("S3_BUCKETNAME"), hash, file, file.(Sizer).Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})

	w.Write([]byte("hehez"))
}
