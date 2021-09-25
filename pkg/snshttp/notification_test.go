package snshttp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotification_ARNShort(t *testing.T) {
	t.Run("standard case", func(t *testing.T) {
		notification := &SNSNotification{
			TopicArn: "arn:aws:sns:us-east-1:123123123:test-hook",
		}

		expected := "test-hook"

		require.Equal(t, expected, notification.ARNShort())
	})

	t.Run("odd case", func(t *testing.T) {
		notification := &SNSNotification{
			TopicArn: "arn:aws:sns:us-east-1:123123123:",
		}

		expected := ""

		require.Equal(t, expected, notification.ARNShort())
	})

}
