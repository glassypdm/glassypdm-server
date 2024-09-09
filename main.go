package main

import (
	"context"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joshtenorio/glassypdm-server/sqlcgen"

	"github.com/joho/godotenv"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
)

func main() {
	ctx := context.Background()
	godotenv.Load()

	clerk.SetKey(os.Getenv("CLERK_SECRETKEY"))

	// TODO url
	db_pool, err := pgxpool.New(ctx, "")
	if err != nil {
		log.Fatal("could not connect to db", "db error", err)
	}
	defer db_pool.Close()

	queries = *sqlcgen.New(db_pool)
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
	r.Get("/client-config", getConfig)

	// TODO protect them
	r.Post("/store/download", GetS3Download)
	r.Post("/store/request", HandleUpload)

	// Clerk-protected routes
	r.Group(func(r chi.Router) {
		r.Use(clerkhttp.WithHeaderAuthorization())
		r.Get("/permission", GetPermission)
		r.Post("/permission", SetPermission)
		r.Post("/commit", CreateCommit)
		r.Get("/commit/select/by-project/{project-id}", GetCommits)
		r.Post("/project", CreateProject)
		r.Get("/project/info", GetProjectInfo)
		r.Get("/project/user", GetProjectsForUser)
		r.Get("/project/status/by-id/{project-id}", GetProjectState)
		r.Post("/team", CreateTeam)
		r.Get("/team", GetTeamForUser)
		r.Get("/team/by-id/{team-id}", getTeamInformation)
		r.Get("/team/by-id/{team-id}/pgroup/list", GetPermissionGroups)
		r.Post("/team/by-id/{team-id}/pgroup/create", CreatePermissionGroup)
		r.Post("/pgroup/map", CreatePGMapping)
		r.Get("/pgroup/info", GetPermissionGroupInfo)
		// remove mapping
		r.Post("/pgroup/add", AddUserToPG)
		r.Post("/pgroup/remove", RemoveUserFromPG)
		// removem member
		// delete permission group
	})

	if os.Getenv("DEBUG") == "1" {
		log.SetLevel(log.DebugLevel)
	}
	log.SetReportCaller(true)
	log.SetReportTimestamp(true)

	port := os.Getenv("PORT")
	log.Info("Listening on localhost", "port", port)
	http.ListenAndServe(":"+port, r)
}
