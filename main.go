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
	r.Get("/projects", getProjectsForUser)
	r.Get("/project", getProjectInfo)

	// protected routes
	r.Group(func(r chi.Router) {
		r.Use(clerkhttp.WithHeaderAuthorization())
		r.Get("/hehez", protectedRoute)
		r.Post("/ingest", handleUpload)
		r.Get("/permission", getPermission)
		r.Post("/permission", setPermission)
		r.Get("/team/members", getTeamMembership)
		r.Post("/commit", commit)
		r.Post("/project", createProject)
		r.Get("/project/new", getNewFiles)
		r.Get("/project/commit", getLatestCommit)
		r.Get("/project/file", getLatestRevision)
		r.Post("/team", createTeam)
	})

	port := os.Getenv("PORT")

	fmt.Println("Listening on localhost:" + port)
	http.ListenAndServe(":"+port, r)
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
