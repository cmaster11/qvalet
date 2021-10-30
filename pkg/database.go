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

const databaseSchemePostgres = "postgres"

// @formatter:off
/// [database-docs]

/*
Examples of configurations:

- TCP connection:

	host: localhost
	port: 5432
	dbName: myDb
	username: hello
	password: world
	options:
		sslmode: disable

- Unix socket connection:

	dbName: myDb
	username: hello
	password: world
	options:
		host: /var/lib/postgresql
*/
type DatabaseConfig struct {
	// Database hostname
	Host string `mapstructure:"host"`

	// Port to use
	Port int `mapstructure:"port"`

	// Database name, e.g. `mydb`
	DbName string `mapstructure:"dbName" validate:"required"`

	// Connection username
	Username *string `mapstructure:"username"`

	// Connection password
	Password *string `mapstructure:"password"`

	// Additional connection options, e.g. `sslmode: disable`
	// See available options at https://bun.uptrace.dev/postgres/#pgdriver
	Options map[string]string `mapstructure:"options"`
}

/// [database-docs]
// @formatter:on

func (config *DatabaseConfig) ParsedUserInfo() *url.Userinfo {
	if config.Username == nil {
		return nil
	}

	if config.Password != nil {
		return url.UserPassword(*config.Username, *config.Password)
	}

	return url.User(*config.Username)
}

func (config *DatabaseConfig) ParsedOptions() string {
	options := make(url.Values)
	for key, val := range config.Options {
		options.Set(key, val)
	}
	return options.Encode()
}

func (config *DatabaseConfig) parsedConnectionURL() *url.URL {
	u := new(url.URL)
	u.Scheme = databaseSchemePostgres
	u.User = config.ParsedUserInfo()

	if config.Port != 0 {
		u.Host = fmt.Sprintf("%s:%d", config.Host, config.Port)
	} else {
		u.Host = config.Host
	}

	u.Path = "/" + config.DbName

	u.RawQuery = config.ParsedOptions()
	return u
}

func (config *DatabaseConfig) parsedLogSafeConnectionURL() string {
	u := config.parsedConnectionURL()
	u.User = nil
	return u.String()
}

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
		return errors.WithMessagef(err, "failed to init migrations for %s in db %s", plugin.Id(), w.config.parsedLogSafeConnectionURL())
	}

	group, err := migrator.Migrate(context.Background())
	if err != nil {
		return errors.WithMessagef(err, "failed to perform migrations for %s in db %s", plugin.Id(), w.config.parsedLogSafeConnectionURL())
	}

	if group.ID == 0 {
		return nil
	}

	logrus.WithField("dsn", w.config.parsedLogSafeConnectionURL()).Infof("migrated plugin %s database to %s", plugin.Id(), group)
	return nil
}

func NewDB(config *DatabaseConfig) (*BunDbWrapper, error) {
	dsn := config.parsedConnectionURL().String()

	// Check if we already have a db for this DSN
	if bunDBIntf, found := databaseConnectionCache.Load(dsn); found {
		return bunDBIntf.(*BunDbWrapper), nil
	}

	sqlDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	bunDB := bun.NewDB(sqlDB, pgdialect.New())

	if logrus.IsLevelEnabled(logrus.DebugLevel) && os.Getenv("QV_VERBOSE_DATABASE") == "true" {
		bunDB.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	}

	if err := bunDB.Ping(); err != nil {
		defer bunDB.Close()
		return nil, errors.WithMessage(err, "failed to connect to database")
	}

	logrus.WithField("dsn", config.parsedLogSafeConnectionURL()).Info("connected to database")

	wrapper := &BunDbWrapper{
		config, bunDB, nil,
	}

	databaseConnectionCache.Store(dsn, wrapper)

	return wrapper, nil
}
