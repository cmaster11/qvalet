package pkg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

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

// # curl "http://localhost:7055/auth/basic" -u myUser:helloBasic
var regexTestCase = regexp.MustCompile(`(?im)^.*?# (?:\[(\d+)(?:,(ERR))?] )?curl "http://localhost:7055/([^"]+)"(.*)$\n(?:.*?(# Expect .+$))?`)
var regexExpectError = regexp.MustCompile(`^# Expect(?: (` + expectPrefixError + `|` + expectPrefixErrorContains + `)) (".+)$`)
var regexExpectOutput = regexp.MustCompile(`^# Expect (.+)$`)
var regexStatus = regexp.MustCompile(`(?m)^STATUS:(\d+)$`)

func TestExamples(t *testing.T) {
	// If we defined any test env file, load it
	{
		envFile := os.Getenv("TEST_ENV_FILE")
		if envFile != "" {
			t.Logf("loading env file %s", envFile)
			require.NoError(t, godotenv.Load(envFile))
		}
	}

	t.Logf("current PATH: %s", os.Getenv("PATH"))

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
			_, router := loadGTE(examplePath)

			listener, _ := net.Listen("tcp", ":0")
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

						var tmpFile string
						if shouldHaveErr {
							tmpFile = fmt.Sprintf("%s/test-%d-err.txt", testTempDir, time.Now().UnixNano())
							command = fmt.Sprintf(`%s -H "X-ERR-FILE: %s"`, command, tmpFile)
							defer os.Remove(tmpFile)
						}

						// Generate and run the test go script for the current test case
						code, err := getCurlToGoCode(command)
						require.NoErrorf(t, err, "curl to go code: %s", code)

						t.Logf("executing go code:\n%s", code)

						rawOutput, result, err := execGoTest(t, code, int(statusCode))
						require.NoErrorf(t, err, "go execution: %v", rawOutput)

						if shouldHaveErr {
							// Check that there is content in the tmp file
							c, err := os.ReadFile(tmpFile)
							require.NoError(t, err)
							require.NotEmpty(t, c)
						}

						t.Log(spew.Sprint("executed go code: %v", result))

						if expect != "" {
							// If we're expecting a specific output, check!
							var errStr string
							if result.Response.Error != nil {
								errStr = *result.Response.Error
							}
							switch expectPrefix {
							case expectPrefixError:
								require.EqualValues(t, expect, strings.TrimSpace(errStr))
							case expectPrefixErrorContains:
								require.Contains(t, strings.TrimSpace(errStr), expect)
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
	defer os.Remove(fileName)

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
		return string(out), result, errors.New(fmt.Sprintf("bad status code %d", result.Status))
	}

	return string(out), result, nil
}

func loadGTE(configPath string) (*GoToExec, *gin.Engine) {
	config := MustLoadConfig(configPath)
	// We rely on outputs to check if tests are successful
	config.Defaults.ReturnOutput = boolPtr(true)
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(gin.ErrorLogger())
	gte := NewGoToExec(config)
	gte.MountRoutes(router)
	return gte, router
}

var visitRegex = regexp.MustCompile(`^config.(.+).yaml$`)
var visitRegexIgnore = regexp.MustCompile(`^config(.*).ignore.yaml$`)

func visit(files *[]string, toTest []string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logrus.Fatal(err)
		}

		name := info.Name()
		if visitRegexIgnore.MatchString(name) {
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
