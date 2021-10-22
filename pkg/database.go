package pkg

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
)

// @formatter:off
/// [database-docs]

type DatabaseConfig struct {
	// Database hostname, defaults to `localhost`
	Host string `mapstructure:"host"`

	// Port to use, defaults to `5432`
	Port int `mapstructure:"port"`

	// Database name, e.g. `mydb`
	DbName string `mapstructure:"dbName" validate:"required"`

	// Connection username
	Username *string `mapstructure:"username"`

	// Connection password
	Password *string `mapstructure:"password"`

	// Additional connection options, e.g. `sslmode: disable`
	Options map[string]string `mapstructure:"options"`
}

func (config *DatabaseConfig) ParsedUserPassDSN() string {
	userDSN := ""
	if config.Username != nil {
		userDSN = *config.Username

		if config.Password != nil {
			userDSN = fmt.Sprintf("%s:%s", userDSN, *config.Password)
		}

		userDSN = fmt.Sprintf("%s@", userDSN)
	}
	return userDSN
}

func (config *DatabaseConfig) ParsedHostname() string {
	hostname := "localhost"
	if config.Host != "" {
		hostname = config.Host
	}
	return hostname
}

func (config *DatabaseConfig) ParsedPort() int {
	port := 5432
	if config.Port != 0 {
		port = config.Port
	}
	return port
}

func (config *DatabaseConfig) ParsedOptions() string {
	options := make(url.Values)
	for key, val := range config.Options {
		options.Set(key, val)
	}
	return options.Encode()
}

func (config *DatabaseConfig) ParsedDSN() string {
	return fmt.Sprintf("postgres://%s%s:%d/%s?%s", config.ParsedUserPassDSN(), config.ParsedHostname(), config.ParsedPort(), config.DbName, config.ParsedOptions())
}

func (config *DatabaseConfig) ParsedLogSafeDSN() string {
	return fmt.Sprintf("%s:%d/%s", config.ParsedHostname(), config.ParsedPort(), config.DbName)
}

/// [database-docs]
// @formatter:on

// We keep a global connection cache, in case a DB is reused
var databaseConnectionCache = new(sync.Map)

// To be invoked on shutdown
func CloseAllDBConnections() {
	databaseConnectionCache.Range(func(key, value interface{}) bool {
		if err := value.(*BunDbWrapper).db.Close(); err != nil {
			logrus.WithError(err).Errorf("failed to close db")
		}
		databaseConnectionCache.Delete(key)
		return true
	})
}

type BunDbWrapper struct {
	config            *DatabaseConfig
	db                *bun.DB
	appliedMigrations []*migrate.Migrations
}

func (w *BunDbWrapper) DB() *bun.DB {
	return w.db
}

// Because we have multiple plugins, and listener, and on init each listener/plugin may want to
// run the same migrations, just keep track of already executed ones in a cache
func (w *BunDbWrapper) ApplyMigrations(plugin PluginConfigNeedsDb) error {
	migrations := plugin.Migrations()
	if migrations == nil {
		return nil
	}

	for _, v := range w.appliedMigrations {
		if v == migrations {
			return nil
		}
	}

	migrator := migrate.NewMigrator(w.DB(), migrations)

	if err := migrator.Init(context.Background()); err != nil {
		return errors.WithMessagef(err, "failed to init migrations for %s in db %s", plugin.Id(), w.config.ParsedLogSafeDSN())
	}

	group, err := migrator.Migrate(context.Background())
	if err != nil {
		return errors.WithMessagef(err, "failed to perform migrations for %s in db %s", plugin.Id(), w.config.ParsedLogSafeDSN())
	}

	if group.ID == 0 {
		return nil
	}

	logrus.WithField("dsn", w.config.ParsedLogSafeDSN()).Infof("migrated plugin %s database to %s", plugin.Id(), group)
	return nil
}

func NewDB(config *DatabaseConfig) (*BunDbWrapper, error) {
	dsn := config.ParsedDSN()

	// Check if we already have a db for this DSN
	if bunDBIntf, found := databaseConnectionCache.Load(dsn); found {
		return bunDBIntf.(*BunDbWrapper), nil
	}

	sqlDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	bunDB := bun.NewDB(sqlDB, pgdialect.New())

	if logrus.IsLevelEnabled(logrus.DebugLevel) && os.Getenv("GTE_VERBOSE_DATABASE") == "true" {
		bunDB.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	}

	if err := bunDB.Ping(); err != nil {
		defer bunDB.Close()
		return nil, errors.WithMessage(err, "failed to connect to database")
	}

	logrus.WithField("dsn", config.ParsedLogSafeDSN()).Info("connected to database")

	wrapper := &BunDbWrapper{
		config, bunDB, nil,
	}

	databaseConnectionCache.Store(dsn, wrapper)

	return wrapper, nil
}
