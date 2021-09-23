module gotoexec

go 1.16

require (
	github.com/Masterminds/goutils v1.1.1
	github.com/Masterminds/sprig/v3 v3.2.1
	github.com/beyondstorage/go-service-azblob/v2 v2.2.0
	github.com/beyondstorage/go-service-cos/v2 v2.2.0
	github.com/beyondstorage/go-service-dropbox/v2 v2.2.0
	github.com/beyondstorage/go-service-fs/v3 v3.3.0
	github.com/beyondstorage/go-service-gcs/v2 v2.2.0
	github.com/beyondstorage/go-service-kodo/v2 v2.2.0
	github.com/beyondstorage/go-service-oss/v2 v2.3.0
	github.com/beyondstorage/go-service-qingstor/v3 v3.2.0
	github.com/beyondstorage/go-service-s3/v2 v2.3.0
	github.com/beyondstorage/go-storage/v4 v4.4.1-0.20210730075750-6e541b87ea46
	github.com/davecgh/go-spew v1.1.1
	github.com/gin-contrib/timeout v0.0.1
	github.com/gin-gonic/gin v1.7.2
	github.com/go-playground/validator/v10 v10.4.1
	github.com/goccy/go-yaml v1.9.2
	github.com/hashicorp/go-retryablehttp v0.7.0
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12
	github.com/jessevdk/go-flags v1.5.0
	github.com/joho/godotenv v1.3.0
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.4.1
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
)

replace github.com/spf13/viper v1.7.1 => github.com/kublr/viper v1.6.3-0.20200316132607-0caa8e000d5b

replace github.com/beyondstorage/go-service-gcs/v2 v2.2.0 => github.com/cmaster11/go-service-gcs/v2 v2.2.1-0.20210816062650-f8d8c7d1c0e1
