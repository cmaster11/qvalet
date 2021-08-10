package pkg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseIfTemplate(t *testing.T) {
	t.Run("simple true", func(t *testing.T) {
		_, err := ParseIfTemplate("test", `true`)
		require.NoError(t, err)
	})

	t.Run("not hackable with }}", func(t *testing.T) {
		_, err := ParseIfTemplate("test", ` true }} hehe {{ end }} {{ if true`)
		require.Error(t, err)
	})

	t.Run("not hackable with }} multiline", func(t *testing.T) {
		_, err := ParseIfTemplate("test", ` true }
} hehe {{ end }
} {{ if true`)
		require.Error(t, err)
	})

	t.Run("multiline", func(t *testing.T) {
		_, err := ParseIfTemplate("test", `
and (
	true
	true
)
`)
		require.NoError(t, err)
	})

}
