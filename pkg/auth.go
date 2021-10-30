package pkg

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"

	"qvalet/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// @formatter:off
/// [auth-docs]

type AuthConfig struct {
	// Api keys for this auth type.
	// Each api key can also be loaded from the environment variables, by
	// using the syntax `ENV{ENV_VAR_NAME}`, e.g.
	//
	// apiKeys:
	// 	 - ENV{MY_PASSWORD}
	//
	ApiKeys []*utils.StringFromEnvVar `mapstructure:"apiKeys" validate:"required"`

	// If true, allows basic HTTP authentication
	BasicAuth bool `mapstructure:"basicAuth"`

	// If true, url query authentication will be allowed
	QueryAuth bool `mapstructure:"queryAuth"`

	// The key to check for in the url query.
	// Defaults to `__qvApiKey` if none is provided
	QueryAuthKey string `mapstructure:"queryAuthKey"`

	// The basic auth HTTP username.
	// Defaults to `qv` if none is provided
	BasicAuthUser string `mapstructure:"basicAuthUser"`

	// If provided, apiKeys will be searched for in these headers
	// E.g. GitLab hooks can authenticate via X-Gitlab-Token
	AuthHeaders []*AuthHeader `mapstructure:"authHeaders" validate:"dive"`
}

type AuthHeader struct {
	// Header name, case-insensitive
	Header string `mapstructure:"header"`

	// If provided, the header content will be compared using this method
	Method AuthHeaderMethod `mapstructure:"method" validate:"authHeaderMethod"`

	// If provided, this is used to alter the incoming header value, where
	// the header value is the current context `.`
	// E.g. for GitHub webhooks, `{{ replace "sha256=" "" . }}` would strip out the
	// initial sha256= prefix GitHub passes to all webhooks
	Transform *Template `mapstructure:"transform"`
}

type AuthHeaderMethod string

const (
	// Simply compares the value of the header with every api key
	AuthHeaderMethodNone AuthHeaderMethod = ""

	// Calculates the payload HMAC-SHA256 hash for each api key,
	// and compares the hash with the value provided in the header.
	AuthHeaderMethodHMACSHA256 AuthHeaderMethod = "hmac-sha256"
)

/// [auth-docs]
// @formatter:on

func verifyAuth(c *gin.Context, authConfigs []*AuthConfig) error {
	if len(authConfigs) == 0 {
		return nil
	}

	// Auth check
	found := false

	// Cache the body data, if needed
	var bodyData []byte

	for _, auth := range authConfigs {

		// Basic HTTP authentication
		if auth.BasicAuth {
			authUser := auth.BasicAuthUser
			if authUser == "" {
				// Default user for basic auth
				authUser = keyAuthDefaultHTTPBasicUser
			}
			// Check if there is any basic auth
			if username, password, ok := c.Request.BasicAuth(); ok {
				if username == authUser {
					for _, apiKey := range auth.ApiKeys {
						if password == apiKey.Value() {
							found = true
							goto afterAuth
						}
					}
				}
			}
		}

		// Url query authentication
		if auth.QueryAuth {
			queryKey := auth.QueryAuthKey
			if queryKey == "" {
				queryKey = keyAuthApiKeyQuery
			}
			apiKeyQuery := c.Query(queryKey)
			for _, apiKey := range auth.ApiKeys {
				if apiKeyQuery == apiKey.Value() {
					found = true
					goto afterAuth
				}
			}
		}

		// Header authentication
		if len(auth.AuthHeaders) > 0 {
			for _, authHeader := range auth.AuthHeaders {
				headerValue := c.GetHeader(authHeader.Header)
				for _, apiKey := range auth.ApiKeys {
					isValid := false

					if authHeader.Transform != nil {
						_headerValue, err := authHeader.Transform.Execute(headerValue)
						if err != nil {
							return errors.WithMessage(err, "failed to execute header template")
						}
						headerValue = _headerValue
					}

					switch authHeader.Method {
					case AuthHeaderMethodNone:
						isValid = headerValue == apiKey.Value()
					case AuthHeaderMethodHMACSHA256:
						if bodyData == nil {
							data, err := c.GetRawData()
							if err != nil {
								return errors.WithMessage(err, "failed to read body data")
							}
							bodyData = data
							// Put the data back for later usage
							c.Request.Body = ioutil.NopCloser(bytes.NewReader(data))
						}

						hmacValue := authHMACSHA256(bodyData, apiKey.Value())
						isValid = headerValue == hmacValue
					default:
						return errors.New("bad header auth method")
					}

					if isValid {
						found = true
						goto afterAuth
					}
				}
			}
		}
	}

afterAuth:

	if !found {
		return errors.New("bad auth")
	}

	return nil
}

func authHMACSHA256(data []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}
