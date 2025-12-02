package db

import (
	"database/sql"
	"fmt"
	"l0/migrations"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

type ZapGooseAdapter struct {
	*zap.Logger
}

func (z *ZapGooseAdapter) Fatalf(format string, v ...any) {
	z.Fatal(fmt.Sprintf(format, v...))
}

func (z *ZapGooseAdapter) Printf(format string, v ...any) {
	z.Info(fmt.Sprintf(format, v...))
}

func RunMigrations(dbConn *sql.DB, logger *zap.Logger) {
	goose.SetLogger(&ZapGooseAdapter{Logger: logger})

	if err := goose.SetDialect("postgres"); err != nil {
		logger.Fatal("Failed to set database dialect", zap.Error(err))
	}

	goose.SetBaseFS(migrations.EmbedFS)

	if err := goose.Up(dbConn, "."); err != nil {
		logger.Fatal("Failed to run migrations", zap.Error(err))
	}

	logger.Info("Database migrations applied successfully")
}
