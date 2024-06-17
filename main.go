package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/joho/godotenv"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
)

func main() {

	godotenv.Load()

	clerk.SetKey(os.Getenv("CLERK_SECRETKEY"))

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://*", "http://*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello"))
	})
	r.Get("/version", getVersion)
	r.Get("/daijin-config", getConfig)

	// custom JWT-protected routes
	r.Group(func(r chi.Router) {
		r.Post("/store", HandleUpload)
	})

	// Clerk-protected routes
	r.Group(func(r chi.Router) {
		r.Use(clerkhttp.WithHeaderAuthorization())
		r.Get("/permission", GetPermission)
		r.Post("/permission", SetPermission)
		//r.Post("/storerequest", HandleStoreRequest)
		r.Post("/commit", CreateCommit)
		r.Post("/project", CreateProject)
		r.Get("/project/info", GetProjectInfo)
		r.Get("/project/user", GetProjectsForUser)
		r.Get("/project/new", GetNewFiles)
		r.Get("/project/commit", GetLatestCommit)
		r.Get("/project/file", GetLatestRevision)
		r.Post("/team", CreateTeam)
		r.Get("/team", GetTeamForUser)
		r.Get("/team/by-id/{teamId}", getTeamInformation)
	})

	port := os.Getenv("PORT")

	fmt.Println("Listening on localhost:" + port)
	http.ListenAndServe(":"+port, r)
}
