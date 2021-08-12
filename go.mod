module gotoexec

go 1.16

require (
	github.com/Masterminds/sprig/v3 v3.2.1
	github.com/davecgh/go-spew v1.1.1
	github.com/gin-gonic/gin v1.7.2
	github.com/go-playground/validator/v10 v10.4.1
	github.com/goccy/go-yaml v1.9.2
	github.com/google/uuid v1.2.0 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12
	github.com/joho/godotenv v1.3.0
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/sys v0.0.0-20210809222454-d867a43fc93e // indirect
	golang.org/x/text v0.3.3 // indirect
)

replace github.com/spf13/viper v1.7.1 => github.com/kublr/viper v1.6.3-0.20200316132607-0caa8e000d5b
