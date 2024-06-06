package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

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
// chunk + filekey (hash)
func handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(20 * (1 << 20)); err != nil { // TODO adjust max memory; currently 20 * (1 << 20) is 20 MB. config it from env
		w.Write([]byte("error"))
		return
	}
	// TODO make sure uploader has permission to upload to at least one project

	file, _, err := r.FormFile("filekey")
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

	s3.PutObject(context.Background(), os.Getenv("S3_BUCKETNAME"), "readAAaame.md", file, file.(Sizer).Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})

	w.Write([]byte("hehez"))
}
