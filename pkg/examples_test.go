package pkg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"gotoexec/pkg/utils"

	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const hackDir = "../hack"
const examplesDir = "../examples"
const curlToGoInit = "curlToGoInit.js"
const testTempDir = "../.test"

const expectPrefixError = "error"
const expectPrefixErrorContains = "error contains"
const expectPrefixErrorHandlerResult = "error handler result"

// # curl "http://localhost:7055/auth/basic" -u myUser:helloBasic
var regexTestCase = regexp.MustCompile(`(?im)^.*?# (?:\[(\d+)(?:,(ERR))?] )?curl "http://localhost:7055/([^"]+)"(.*)$\n(?:.*?(# Expect .+$))?`)
var regexExpectError = regexp.MustCompile(`^# Expect(?: (` + expectPrefixError + `|` + expectPrefixErrorContains + `|` + expectPrefixErrorHandlerResult + `)) (".+)$`)
var regexExpectOutput = regexp.MustCompile(`^# Expect (.+)$`)

func TestExamples(t *testing.T) {
	// If we defined any test env file, load it
	{
		envFile := os.Getenv("TEST_ENV_FILE")
		if envFile != "" {
			t.Logf("loading env file %s", envFile)
			require.NoError(t, godotenv.Load(envFile))
		}
	}

	// t.Logf("current PATH: %s", os.Getenv("PATH"))

	// Check if we have only a subset of tests to run
	var toTest []string
	toTestStr := os.Getenv("TESTS")
	if toTestStr != "" {
		toTest = strings.Split(toTestStr, ",")
	}

	// parallel := os.Getenv("TEST_SERIAL") != "true"

	/*
		Read every example, and execute the provided tests.
	*/
	var examplesFiles []string

	err := filepath.Walk(examplesDir, visit(&examplesFiles, toTest))
	require.NoError(t, err)
	for _, examplePath := range examplesFiles {
		t.Run(fmt.Sprintf("example-%s", examplePath), func(t *testing.T) {
			listener, _ := net.Listen("tcp", ":0")

			router := loadGTE(t, examplePath, listener)

			addr := listener.Addr().String()
			go http.Serve(listener, router)
			defer listener.Close()

			t.Logf("running at %s", addr)

			/*
				Read the file to find testing cases
			*/
			contentBytes, err := ioutil.ReadFile(examplePath)
			require.NoError(t, err)
			content := string(contentBytes)

			wg := sync.WaitGroup{}
			testCases := regexTestCase.FindAllString(content, -1)
			for idx, testCase := range testCases {
				wg.Add(1)
				fn := func() {
					t.Run(fmt.Sprintf("case-%d", idx), func(t *testing.T) {
						defer wg.Done()
						match := regexTestCase.FindStringSubmatch(testCase)

						statusCodeStr := match[1]
						shouldHaveErr := match[2] == "ERR"
						path := match[3]
						args := match[4]
						expectString := match[5]

						var expectPrefix = ""
						var expect = ""
						if expectString != "" {
							if match := regexExpectError.FindStringSubmatch(expectString); match != nil {
								expectPrefix = match[1]
								expect = match[2]
							} else if match := regexExpectOutput.FindStringSubmatch(expectString); match != nil {
								expect = match[1]
							} else {
								t.Fatalf("invalid expect string %s for test %s", expectString, testCase)
							}
						}

						if statusCodeStr == "" {
							statusCodeStr = "200"
						}
						statusCode, err := strconv.ParseInt(statusCodeStr, 10, 32)
						require.NoError(t, err)

						if expect != "" {
							// Internally de-JSONize expect
							// Usually written as "Hello\nWorld"
							var expectStr string
							require.NoError(t, json.Unmarshal([]byte(expect), &expectStr))
							expect = expectStr
						}

						t.Logf("executing test case %s", testCase)

						command := fmt.Sprintf(`curl "http://%s/%s" %s`, addr, path, args)

						// Generate and run the test go script for the current test case
						code, err := getCurlToGoCode(command)
						require.NoErrorf(t, err, "curl to go code: %s", code)

						if os.Getenv("LOG_GO_CODE") == "true" {
							t.Logf("executing go code:\n%s", code)
						}

						rawOutput, result, err := execGoTest(t, code, int(statusCode))
						require.NoErrorf(t, err, "go execution: %v", rawOutput)

						if shouldHaveErr {
							require.NotNil(t, result.Response.ErrorHandlerResult)
							require.NotNil(t, result.Response.ErrorHandlerResult.Storage)
							require.NotEmpty(t, result.Response.ErrorHandlerResult.Storage.Path)
						}

						t.Log(spew.Sprint("executed go code: %v", result))

						if expect != "" {
							// If we're expecting a specific output, check!
							var errStr string
							if result.Response.Error != nil {
								errStr = *result.Response.Error
							}
							var errHandlerResult string
							if result.Response.ErrorHandlerResult != nil {
								errHandlerResult = result.Response.ErrorHandlerResult.Output
							}
							switch expectPrefix {
							case expectPrefixError:
								require.EqualValues(t, expect, strings.TrimSpace(errStr))
							case expectPrefixErrorContains:
								require.Contains(t, strings.TrimSpace(errStr), expect)
							case expectPrefixErrorHandlerResult:
								require.EqualValues(t, expect, strings.TrimSpace(errHandlerResult))
							case "":
								require.EqualValues(t, expect, strings.TrimSpace(result.Response.Output))
							default:
								t.Fatalf("invalid expect prefix %s in test %s", expectPrefix, testCase)
							}
						}
					})
				}

				// if parallel {
				// go fn()
				// } else {
				fn()
				// }
			}
			wg.Wait()

			/*
				w := httptest.NewRecorder()
					req, _ := http.NewRequest("GET", "/ping", nil)
					router.ServeHTTP(w, req)

					assert.Equal(t, 200, w.Code)
					assert.Equal(t, "pong", w.Body.String())
			*/
		})
	}

}

func getCurlToGoCode(curl string) (string, error) {
	cmd := exec.Command("node", curlToGoInit, curl)
	cmd.Dir = hackDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

type execGoTestResultRaw struct {
	Output string `json:"output"`
	Status int    `json:"status"`
}

type execGoTestResult struct {
	Response ListenerResponse
	Status   int
}

func execGoTest(t *testing.T, code string, expectedStatus int) (string, *execGoTestResult, error) {
	// Save the test to a temporary file, and execute
	fileName := fmt.Sprintf("%s/test-%d.go", testTempDir, time.Now().UnixNano())
	if err := ioutil.WriteFile(fileName, []byte(code), 0777); err != nil {
		return "", nil, err
	}

	t.Logf("written test go file %s", fileName)

	out, err := exec.Command("goimports", "-w", fileName).CombinedOutput()
	if err != nil {
		return string(out), nil, err
	}

	cmd := exec.Command("go", "run", fileName)
	out, err = cmd.CombinedOutput()

	resultRaw := &execGoTestResultRaw{}
	_ = json.Unmarshal(out, resultRaw)

	result := &execGoTestResult{
		Status: resultRaw.Status,
	}

	_ = json.Unmarshal([]byte(resultRaw.Output), &result.Response)

	if err != nil {
		if result.Status == expectedStatus {
			return string(out), result, nil
		}

		return string(out), result, err
	}

	if result.Status != expectedStatus {
		return string(out), result, errors.Errorf("bad status code %d", result.Status)
	}
	defer os.Remove(fileName)

	return string(out), result, nil
}

var regexDefaults = regexp.MustCompile(`\[DEFAULTS=([^]]+)]`)
var regexPart = regexp.MustCompile(`\[PART=([^]]+)]`)

func loadGTE(t *testing.T, configPath string, listener net.Listener) *gin.Engine {
	os.Setenv("GTE_TEST_URL", listener.Addr().String())

	if os.Getenv("GTE_VERBOSE") == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}

	// We need to pre-read the config file, to find out if we need to
	// e.g. use a defaults file too
	content, err := ioutil.ReadFile(configPath)
	require.NoError(t, err)
	var defaults *ListenerConfig
	{
		// Find the FIRST defaults
		if match := regexDefaults.FindStringSubmatch(string(content)); match != nil {
			filename := match[1]
			if !path.IsAbs(filename) {
				// Take the file name relative to the config
				filename = filepath.Join(filepath.Dir(configPath), filename)
			}
			t.Logf("using defaults %s", filename)
			_defaults, err := LoadDefaults(filename)
			require.NoError(t, err)
			defaults = _defaults
		}
	}

	var configs []*Config

	{
		config, err := LoadConfig(configPath)
		require.NoError(t, err)
		configs = append(configs, config)
	}

	// Also, check for additional configs to load
	{
		if lines := regexPart.FindAllStringSubmatch(string(content), -1); lines != nil {
			for _, match := range lines {
				filename := match[1]
				if !path.IsAbs(filename) {
					// Take the file name relative to the config
					filename = filepath.Join(filepath.Dir(configPath), filename)
				}

				t.Logf("using additional config %s", filename)
				config, err := LoadConfig(filename)
				require.NoError(t, err)
				configs = append(configs, config)
			}
		}
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(gin.ErrorLogger())

	for _, config := range configs {
		config.Debug = true

		if defaults != nil {
			newDefaults, err := MergeListenerConfig(defaults, &config.Defaults)
			require.NoError(t, err)
			config.Defaults = *newDefaults
		}

		// NOTE: This is needed for tests to succeed!
		config.Defaults.Return = []ReturnKey{ReturnKeyAll}

		require.NoError(t, utils.Validate.Struct(config))
		MountRoutes(router, config)
	}

	return router
}

var visitRegex = regexp.MustCompile(`^config.(.+).yaml$`)
var visitRegexIgnore = regexp.MustCompile(`^config(.*).ignore.yaml$`)
var visitRegexNoTest = regexp.MustCompile(`^config(.*).notest.yaml$`)

func visit(files *[]string, toTest []string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logrus.Fatal(err)
		}

		name := info.Name()
		if visitRegexIgnore.MatchString(name) {
			return nil
		}
		if visitRegexNoTest.MatchString(name) {
			return nil
		}

		match := visitRegex.FindStringSubmatch(name)
		if match == nil {
			return nil
		}

		if len(toTest) > 0 {
			find := match[1]
			for _, v := range toTest {
				if v == find {
					*files = append(*files, path)
					return nil
				}
			}
		} else {
			*files = append(*files, path)
		}
		return nil
	}
}
