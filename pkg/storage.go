package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/goutils"
	"github.com/goccy/go-yaml"

	// Add fs support
	_ "github.com/beyondstorage/go-service-fs/v3"
	"github.com/beyondstorage/go-storage/v4/services"
	"github.com/beyondstorage/go-storage/v4/types"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"

	_ "github.com/beyondstorage/go-service-azblob/v2"
	_ "github.com/beyondstorage/go-service-cos/v2"
	_ "github.com/beyondstorage/go-service-dropbox/v2"
	_ "github.com/beyondstorage/go-service-gcs/v2"
	_ "github.com/beyondstorage/go-service-kodo/v2"
	_ "github.com/beyondstorage/go-service-oss/v2"
	_ "github.com/beyondstorage/go-service-qingstor/v3"
	_ "github.com/beyondstorage/go-service-s3/v2"
)

// @formatter:off
/// [storage-docs]

type StorageConfig struct {
	// Connection string for the storage service where payloads will be stored.
	// Ref: https://beyondstorage.io/docs/go-storage/services/index
	Conn string `mapstructure:"conn" validate:"required"`

	// What to store? Can be a comma-separated mix of:
	// - all: store everything
	// - args: store every request's args
	// - command: store every request's executed command details, its args and env vars
	// - env: store every request's executed command env vars
	// - output: store every executed command result
	Store []StoreKey `mapstructure:"store" validate:"required,dive,storageStoreKey"`

	// If true, stores the payload as YAML instead of JSON, improving human readability
	// of the contents
	AsYAML bool `mapstructure:"asYAML"`
}

/// [storage-docs]
// @formatter:on

type StoreKey string

const (
	StoreKeyAll     StoreKey = "all"
	StoreKeyArgs    StoreKey = "args"
	StoreKeyCommand StoreKey = "command"
	StoreKeyEnv     StoreKey = "env"
	StoreKeyOutput  StoreKey = "output"
)

func storeKeyContains(values []StoreKey, search StoreKey) bool {
	for _, val := range values {
		if val == search {
			return true
		}
	}
	return false
}

func (c *StorageConfig) StoreArgs() bool {
	return storeKeyContains(c.Store, StoreKeyArgs) || storeKeyContains(c.Store, StoreKeyAll)
}
func (c *StorageConfig) StoreCommand() bool {
	return storeKeyContains(c.Store, StoreKeyCommand) || storeKeyContains(c.Store, StoreKeyAll)
}
func (c *StorageConfig) StoreEnv() bool {
	return storeKeyContains(c.Store, StoreKeyEnv) || storeKeyContains(c.Store, StoreKeyAll)
}
func (c *StorageConfig) StoreOutput() bool {
	return storeKeyContains(c.Store, StoreKeyOutput) || storeKeyContains(c.Store, StoreKeyAll)
}

func init() {
	if err := Validate.RegisterValidation("storageStoreKey", func(fl validator.FieldLevel) bool {
		key := StoreKey(fl.Field().String())
		return key == StoreKeyAll || key == StoreKeyArgs || key == StoreKeyCommand || key == StoreKeyEnv || key == StoreKeyOutput
	}); err != nil {
		logrus.Fatal("failed to register authHeaderMethod validator")
	}
}

func GetStoragerFromString(connectionString string) (types.Storager, error) {
	return services.NewStoragerFromString(connectionString)
}

type StorageEntry struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func storePayload(
	listener *CompiledListener,
	toStore map[string]interface{},
) *StorageEntry {
	log := listener.log

	refRoute := listener.route
	suffix := ""
	if listener.isErrorHandler {
		refRoute = listener.sourceRoute
		suffix = "-error"
	}

	routePrefix := regexListenerRouteCleaner.ReplaceAllString(refRoute, "_")
	nowMs := time.Now().UnixNano() / int64(time.Millisecond)
	rand, _ := goutils.RandomAlphaNumeric(8)
	extension := "json"
	if listener.config.Storage.AsYAML {
		extension = "yaml"
	}
	path := fmt.Sprintf("%s-%d%s-%s.%s", routePrefix, nowMs, suffix, rand, extension)

	var b []byte

	if listener.config.Storage.AsYAML {
		_b, err := yaml.Marshal(toStore)
		if err != nil {
			log.WithError(err).Error("failed to marshal (yaml) payload for storage")
		}
		b = _b
	} else {
		_b, err := json.Marshal(toStore)
		if err != nil {
			log.WithError(err).Error("failed to marshal (json) payload for storage")
		}
		b = _b
	}

	size := int64(len(b))
	_, err := listener.storager.Write(path, bytes.NewBuffer(b), size)
	if err != nil {
		log.WithError(err).Error("failed to store payload")
		return nil
	}

	if listener.config.LogStorage() {
		log.WithField("path", path).WithField("size", size).Info("stored payload")
	} else {
		log.WithField("path", path).WithField("size", size).Debug("stored payload")
	}
	return &StorageEntry{
		path,
		size,
	}
}
