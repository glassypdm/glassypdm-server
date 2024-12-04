package dal

import (
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joshtenorio/glassypdm-server/internal/sqlcgen"
)

var Queries sqlcgen.Queries
var DbPool *pgxpool.Pool
