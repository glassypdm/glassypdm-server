package main

import (
	"context"
	_ "embed"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/jwtauth/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joshtenorio/glassypdm-server/internal/dal"
	"github.com/joshtenorio/glassypdm-server/internal/observer"
	"github.com/joshtenorio/glassypdm-server/internal/project"
	"github.com/joshtenorio/glassypdm-server/internal/sqlcgen"
	"github.com/posthog/posthog-go"

	"github.com/joho/godotenv"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
)

//go:embed schema.sql
var Ddl string

func main() {
	ctx := context.Background()
	godotenv.Load()

	if os.Getenv("DEBUG") == "1" {
		log.SetLevel(log.DebugLevel)
	}
	log.SetReportCaller(true)
	log.SetReportTimestamp(true)

	clerk.SetKey(os.Getenv("CLERK_SECRETKEY"))
	PSQLUser := os.Getenv("PSQL_USERNAME")
	PSQLPass := os.Getenv("PSQL_PASSWORD")
	PSQLUrl := os.Getenv("PSQL_URL")
	PSQLDatabase := os.Getenv("PSQL_DATABASE")
	PSQLFullURL := os.Getenv("PSQL_FULL_URL")
	PostHogAPIKey := os.Getenv("POSTHOG_API_KEY")
	if PSQLUser == "" || PSQLPass == "" || PSQLUrl == "" || PSQLDatabase == "" || PSQLFullURL == "" || PostHogAPIKey == "" {
		log.Fatal("Missing a database environment")
	}
	var err error

	observer.PostHogClient, err = posthog.NewWithConfig(PostHogAPIKey, posthog.Config{Endpoint: "https://us.i.posthog.com"})
	if err != nil {
		log.Fatal("could not connect to posthog")
	}
	defer observer.PostHogClient.Close()

	url := PSQLFullURL
	//log.Debug("PSQL url", "url", url)
	dal.DbPool, err = pgxpool.New(ctx, url)
	if err != nil {
		log.Fatal("could not connect to db", "db error", err)
	}
	defer dal.DbPool.Close()

	//log.Debug("ddl", "ddl", ddl)
	_, err = dal.DbPool.Exec(ctx, Ddl)
	if err != nil {
		log.Fatal("error executing ddl", "db error", err)
	}

	dal.Queries = *sqlcgen.New(dal.DbPool)
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
		observer.PostHogClient.Enqueue(posthog.Capture{
			DistinctId: "test-user",
			Event:      "test-snippet",
		})
		w.Write([]byte("Hello"))
	})
	r.Get("/version", getVersion)
	r.Get("/client-config", getConfig)

	// TODO protect them
	r.Post("/store/download", GetS3Download)
	r.Post("/store/request", HandleUpload)
	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(project.TokenAuth))
		r.Use(jwtauth.Authenticator(project.TokenAuth))
	})

	// Clerk-protected routes
	r.Group(func(r chi.Router) {
		r.Use(clerkhttp.WithHeaderAuthorization())
		r.Get("/permission", GetPermission)
		r.Post("/permission", SetPermission)
		r.Post("/commit", CreateCommit)
		r.Get("/commit/select/by-project/{project-id}", GetCommits)
		r.Get("/commit/by-id/{commit-id}", GetCommitInformation)
		r.Post("/project", CreateProject)
		r.Get("/project/info", GetProjectInfo)
		r.Get("/project/commit", RouteGetProjectCommit)
		r.Get("/project/user", GetProjectsForUser)
		r.Get("/project/latest", GetProjectLatestCommit) // TODO return more than just commit id
		//r.Post("/project/restore", RouteProjectRestore)
		r.Get("/project/status/by-id/{project-id}", GetProjectState) // TODO remove after v0.7.2 is released
		r.Get("/project/status/by-id/{project-id}/{commit-no}", GetProjectState)
		//r.Get("/project/{project-id}/store", project.RouteStoreJWTRequest)
		r.Post("/team", CreateTeam)
		r.Get("/team", GetTeamForUser)
		r.Get("/team/by-id/{team-id}", getTeamInformation)
		r.Get("/team/by-name/{team-name}", getTeamInformationByName)
		r.Get("/team/basic/by-id/{team-id}", GetBasicTeamInfo)
		r.Get("/team/by-id/{team-id}/pgroup/list", GetPermissionGroups)
		r.Post("/team/by-id/{team-id}/pgroup/create", CreatePermissionGroup)
		r.Post("/pgroup/map", CreatePGMapping)
		r.Get("/pgroup/info", GetPermissionGroupInfo)
		r.Get("/team/by-id/{team-id}/pgroups/{user-id}", GetPermissionGroupForUser)
		r.Get("/team/by-id/{team-id}/pgroups", GetPermissionGroupTeamInfo)
		// remove mapping
		r.Post("/pgroup/add", AddUserToPG)
		r.Post("/pgroup/remove", RemoveUserFromPG)
		// delete permission group
	})

	port := os.Getenv("PORT")
	log.Info("Listening on localhost", "port", port)
	http.ListenAndServe(":"+port, r)

}
