package pkg

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testBackoff struct {
	Str string
	Exp time.Duration
}

func TestTplBackoffCompile(t *testing.T) {

	// tplBackoff(valInitialInterval reflect.Value, valMultiplier reflect.Value, valMaxInterval reflect.Value, valMaxElapsed reflect.Value)
	tests := []testBackoff{
		// Initial interval
		testBackoff{`(duration 10)`, 15 * time.Second},
		testBackoff{`"10s"`, 15 * time.Second},

		// Multiplier
		testBackoff{`"10s" 1`, 10 * time.Second},
		testBackoff{`"10s" 2`, 20 * time.Second},

		// Max interval
		testBackoff{`"10s" 10 "30s"`, 30 * time.Second},
	}

	for _, test := range tests {
		t.Logf("running test %+v", test)

		tplStr := fmt.Sprintf(`{{ backoff %s }}`, test.Str)
		tpl, err := ParseTemplate("test", tplStr)
		require.NoError(t, err, "failed to parse template %s", tplStr)

		out, err := tpl.Execute(nil)
		require.NoErrorf(t, err, "failed to execute template %s", tpl.originalText)

		out = strings.TrimSpace(out)

		d, err := time.ParseDuration(out)
		require.NoErrorf(t, err, "failed to parse template result to duration %s", out)

		require.Equal(t, test.Exp, d)
	}
}
