package pkg

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/ioutil"

	"github.com/gin-gonic/gin"
)

func (gte *GoToExec) verifyAuth(c *gin.Context, listener *CompiledListener) error {
	if len(listener.config.Auth) == 0 {
		return nil
	}

	// Auth check
	found := false

	// Cache the body data, if needed
	var bodyData []byte

	for _, auth := range listener.config.Auth {

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
						if password == apiKey {
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
				if apiKeyQuery == apiKey {
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

					switch authHeader.Method {
					case AuthHeaderMethodNone:
						isValid = headerValue == apiKey
					case AuthHeaderMethodHMACSHA256:
						if bodyData == nil {
							data, err := c.GetRawData()
							if err != nil {
								return errors.New("failed to read body data")
							}
							bodyData = data
							// Put the data back for later usage
							c.Request.Body = ioutil.NopCloser(bytes.NewReader(data))
						}

						hmacValue := authHMACSHA256(bodyData, apiKey)
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
