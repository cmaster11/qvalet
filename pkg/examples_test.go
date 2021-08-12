package pkg

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const hackDir = "../hack"
const examplesDir = "../examples"
const curlToGoInit = "curlToGoInit.js"
const testTempDir = "../.test"

// # curl "http://localhost:7055/auth/basic" -u myUser:helloBasic
var regexTestCase = regexp.MustCompile(`(?im)^.*?# (?:\[(\d+)(?:,(ERR))?] )?curl "http://localhost:7055/([^"]+)"(.*)$`)
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
			for _, testCase := range testCases {
				wg.Add(1)
				go t.Run(fmt.Sprintf("case-%s", testCase), func(t *testing.T) {
					defer wg.Done()
					match := regexTestCase.FindStringSubmatch(testCase)

					statusCode := match[1]
					shouldHaveErr := match[2] == "ERR"
					path := match[3]
					args := match[4]

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

					result, err := execGoTest(t, code, statusCode)
					require.NoErrorf(t, err, "go execution: %s", result)

					if shouldHaveErr {
						// Check that there is content in the tmp file
						c, err := os.ReadFile(tmpFile)
						require.NoError(t, err)
						require.NotEmpty(t, c)
					}

					t.Logf("executed go code: %s", result)
				})
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

func execGoTest(t *testing.T, code string, expectedStatus string) (string, error) {
	// Save the test to a temporary file, and execute
	fileName := fmt.Sprintf("%s/test-%d.go", testTempDir, time.Now().UnixNano())
	if err := ioutil.WriteFile(fileName, []byte(code), 0777); err != nil {
		return "", err
	}
	defer os.Remove(fileName)

	t.Logf("written test go file %s", fileName)

	out, err := exec.Command("goimports", "-w", fileName).CombinedOutput()
	if err != nil {
		return string(out), err
	}

	cmd := exec.Command("go", "run", fileName)
	out, err = cmd.CombinedOutput()
	if err != nil {
		outCode := regexStatus.FindStringSubmatch(string(out))
		if outCode != nil && outCode[1] == expectedStatus {
			return string(out), nil
		}

		return string(out), err
	}
	return string(out), nil
}

func loadGTE(configPath string) (*GoToExec, *gin.Engine) {
	config := MustLoadConfig(configPath)
	config.Defaults.LogArgs = boolPtr(true)
	config.Defaults.LogCommand = boolPtr(true)
	config.Defaults.LogOutput = boolPtr(true)
	config.Defaults.ReturnOutput = boolPtr(true)
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(gin.ErrorLogger())
	gte := NewGoToExec(config)
	gte.MountRoutes(router)
	return gte, router
}

var visitRegex = regexp.MustCompile(`^config.(.+).yaml$`)
var visitRegexIgnore = regexp.MustCompile(`^config.(.+).ignore.yaml$`)

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
