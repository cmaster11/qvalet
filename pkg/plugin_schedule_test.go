package pkg

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseParamTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		Value    string
		Expected time.Time
	}{
		{"3s", now.Add(3 * time.Second)},
		{"2h", now.Add(2 * time.Hour)},
		{"1634197814", time.Unix(1634197814, 0)},
		{"1634197814123", time.UnixMilli(1634197814123)},
		{"2021-10-23T05:18:37+00:00", time.Unix(1634966317, 0)},
	}

	for _, test := range tests {
		parsed, err := parseParamTime(test.Value, &now)
		require.NoError(t, err, "parse %s", test.Value)
		require.Equal(t, test.Expected.UnixMilli(), parsed.UnixMilli(), "parse %s", test.Value)
	}
}
