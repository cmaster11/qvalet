package pkg

import (
	// Add fs support
	_ "github.com/beyondstorage/go-service-fs/v3"
	"github.com/beyondstorage/go-storage/v4/services"
	"github.com/beyondstorage/go-storage/v4/types"

	_ "github.com/beyondstorage/go-service-azblob/v2"
	_ "github.com/beyondstorage/go-service-cos/v2"
	_ "github.com/beyondstorage/go-service-dropbox/v2"
	_ "github.com/beyondstorage/go-service-gcs/v2"
	_ "github.com/beyondstorage/go-service-kodo/v2"
	_ "github.com/beyondstorage/go-service-oss/v2"
	_ "github.com/beyondstorage/go-service-qingstor/v3"
	_ "github.com/beyondstorage/go-service-s3/v2"
)

func GetStoragerFromString(connectionString string) (types.Storager, error) {
	return services.NewStoragerFromString(connectionString)
}
