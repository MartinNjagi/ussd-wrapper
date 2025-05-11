package connections

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

func DbInstance() *sql.DB {
	return newDbInstance(os.Getenv("database_host"))
}

func DbInstanceSlave() *sql.DB {
	return newDbInstance(os.Getenv("database_host_read"))
}

func newDbInstance(host string) *sql.DB {
	username := os.Getenv("database_username")
	password := os.Getenv("database_password")
	dbname := os.Getenv("database_name")
	port := os.Getenv("database_port")

	dbURI := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&multiStatements=true",
		username, password, host, port, dbname)

	db, err := otelsql.Open("mysql", dbURI,
		otelsql.WithAttributes(semconv.DBSystemMySQL),
		otelsql.WithDBName(dbname),
	)
	checkErrFatal(err)

	// Connection pool settings
	idle := fallbackEnvInt("database_idle_connection", 5)
	maxConn := fallbackEnvInt("database_max_connection", 10)
	lifetime := fallbackEnvInt("database_connection_lifetime", 60)

	db.SetMaxIdleConns(idle)
	db.SetMaxOpenConns(maxConn)
	db.SetConnMaxLifetime(time.Second * time.Duration(lifetime))
	db.SetConnMaxIdleTime(time.Second * time.Duration(lifetime))

	otelsql.ReportDBStatsMetrics(db)

	err = db.Ping()
	checkErrFatal(err)

	return db
}

func fallbackEnvInt(envKey string, fallback int) int {
	val, err := strconv.Atoi(os.Getenv(envKey))
	if err != nil {
		return fallback
	}
	return val
}

func checkErrFatal(err error) {
	if err != nil {
		log.Fatalf("DB ERROR: %s", err.Error())
	}
}
