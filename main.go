package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/joho/godotenv"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
)

type Sizer interface {
	Size() int64
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(20 * (1 << 20)); err != nil { // TODO adjust max memory; currently 20 * (1 << 20) is 20 MB
		w.Write([]byte("error"))
		return
	}

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

func main() {

	godotenv.Load()

	clerk.SetKey(os.Getenv("CLERK_SECRETKEY"))

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://*", "http://*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, // TODO technically only need get and post
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello Worlda!"))
	})
	r.Get("/version", getVersion)
	r.Get("/daijin-config", getConfig)

	// protected routes
	r.Group(func(r chi.Router) {
		r.Use(clerkhttp.WithHeaderAuthorization())
		r.Get("/hehez", protectedRoute)
		r.Post("/ingest", handleUpload)
		r.Get("/projects", getProjectsForUser)
	})

	port := os.Getenv("PORT")

	fmt.Println("Listening on localhost:" + port)
	http.ListenAndServe(":"+port, r)
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

func protectedRoute(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	fmt.Fprintf(w, `{"user_id": "%s"}`, claims.Subject)
}
