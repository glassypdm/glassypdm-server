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
	if PSQLUser == "" || PSQLPass == "" || PSQLUrl == "" || PSQLDatabase == "" {
		log.Fatal("Missing a database environment")
	}
	var err error

	url := "postgresql://" + PSQLUser + ":" + PSQLPass + "@" + PSQLUrl + "/" + PSQLDatabase + "?sslmode=require"
	log.Debug("PSQL url", "url", url)
	db_pool, err = pgxpool.New(ctx, url)
	if err != nil {
		log.Fatal("could not connect to db", "db error", err)
	}
	defer db_pool.Close()

	//log.Debug("ddl", "ddl", ddl)
	_, err = db_pool.Exec(ctx, ddl)
	if err != nil {
		// TODO should this be fatal or error?
		log.Fatal("error executing ddl", "db error", err)
	}

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
		r.Get("/team/basic/by-id/{team-id}", GetBasicTeamInfo)
		r.Get("/team/by-id/{team-id}/pgroup/list", GetPermissionGroups)
		r.Post("/team/by-id/{team-id}/pgroup/create", CreatePermissionGroup)
		r.Post("/pgroup/map", CreatePGMapping)
		r.Get("/pgroup/info", GetPermissionGroupInfo)
		r.Get("/team/by-id/{team-id}/pgroups", GetPermissionGroupTeamInfo)
		// remove mapping
		r.Post("/pgroup/add", AddUserToPG)
		r.Post("/pgroup/remove", RemoveUserFromPG)
		// removem member
		// delete permission group
	})

	port := os.Getenv("PORT")
	log.Info("Listening on localhost", "port", port)
	http.ListenAndServe(":"+port, r)
}
