package pkg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFirstBoolPtr(t *testing.T) {
	tr := true
	fa := false
	pTr := &tr
	pFa := &fa

	tests := []struct {
		exp *bool
		val *bool
	}{
		{pTr, firstBoolPtr(pTr)},
		{pFa, firstBoolPtr(pFa)},
		{nil, firstBoolPtr()},
		{pTr, firstBoolPtr(nil, pTr)},
		{pFa, firstBoolPtr(nil, pFa)},
		{pTr, firstBoolPtr(pTr, pFa)},
		{pFa, firstBoolPtr(pFa, pTr)},
	}

	for idx, test := range tests {
		require.Equal(t, test.exp, test.val, "test %d", idx)
	}
}
