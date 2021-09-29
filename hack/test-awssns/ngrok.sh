#!/usr/bin/env bash
set -Eeumo pipefail
DIR=$(
  cd "$(dirname "$0")"
  pwd -P
)

if [[ -n "${DEBUG:-}" ]]; then
  set -x
fi

if ! command -v ngrok; then
  echo "Missing ngrok"
  exit 1
fi

DEV_SSH_ROOT="${DEV_SSH_ROOT:-.}"
AWS_PROFILE="${AWS_PROFILE:-default}"
AWS_SNS_ARN=${AWS_SNS_ARN:-$(cat "$DEV_SSH_ROOT/aws/gte-test-sns-arn.generated")}
AWS_REGION=${AWS_REGION:-$(cat "$DEV_SSH_ROOT/aws/aws-region.generated")}

AWS="aws --region $AWS_REGION"

# Start ngrok in the background, and make sure to kill it on exit
NGROK_PID=
GTE_PID=
SUBSCRIPTION_ARN=

trap cleanup err exit
cleanup() {
  echo "Cleaning up..."

  if [[ -n "$NGROK_PID" ]]; then
    kill -9 "$NGROK_PID" || true
  fi

  if [[ -n "$GTE_PID" ]]; then
    kill -9 "$GTE_PID" || true
  fi

  if [[ -n "$SUBSCRIPTION_ARN" ]]; then
    $AWS sns unsubscribe --subscription-arn "$SUBSCRIPTION_ARN" || true
  fi
}

nohup ngrok http 7055 > nohup-ngrok.log 2>&1 &
NGROK_PID=$!

# Try to get the exposed ngrok URL
HTTPS_URL=

n=0
until [ "$n" -ge 10 ]; do
  HTTPS_URL=$(curl -sS "http://localhost:4040/api/tunnels" --max-time 3 | jq -r -M '.tunnels | .[] | select(.proto == "https") | .public_url' || true)
  if [[ -n "$HTTPS_URL" ]]; then
    break
  fi

  n=$((n + 1))
  sleep 1
done

if [[ -z "$HTTPS_URL" ]]; then
  echo "Unable to set up remote HTTPS endpoint"
  exit 1
fi

# Run gte
TMP_BIN=$(mktemp)
(
  echo "Building binary..."
  cd "$DIR/../.."
  go build -o "$TMP_BIN" ./cmd
  chmod +x "$TMP_BIN"
)
nohup "$TMP_BIN" --config "$DIR/../../examples/config.plugin.awssns.yaml" > nohup-gte.log 2>&1 &
GTE_PID=$!

# Verify that gte is healthy
(
  STATUS=
  n=0
  until [ "$n" -ge 10 ]; do
    STATUS=$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:7055/healthz" || true)
    if [[ "$STATUS" == "200" ]]; then
      break
    fi

    n=$((n + 1))
    sleep 1
  done

  if [[ "$STATUS" != "200" ]]; then
    echo "go-to-exec is not healthy"
    exit 1
  fi
)

echo "Creating SNS subscription..."
# Create an SNS subscription
SNS_RESULT=$(
  $AWS sns subscribe \
    --topic-arn "$AWS_SNS_ARN" \
    --protocol https \
    --notification-endpoint "$HTTPS_URL/hello/sns" \
    --return-subscription-arn
)

SUBSCRIPTION_ARN=$(echo "$SNS_RESULT" | jq -r '.SubscriptionArn')

echo "Waiting for SNS subscription confirmation..."
# Wait until subscription is confirmed
CONFIRMED=false
x=1
while [ $x -le 10 ]; do
  CHECK_RESULT=$($AWS sns get-subscription-attributes \
    --subscription-arn "$SUBSCRIPTION_ARN")
  if [[ "$(echo "$CHECK_RESULT" | jq -r '.PendingConfirmation')" != "true" ]]; then
    CONFIRMED=true
    break
  fi

  sleep 1
  x=$(($x + 1))
done

if [[ "$CONFIRMED" == "false" ]]; then
  echo "SNS subscription not confirmed (timeout)"
  exit 1
fi

echo "Testing SNS subscription..."
# Test by generating a message and expecting the dump file to contain the right text
DATE=$(date +%s%N)
FILENAME="/tmp/dump_aws_sns_message"

$AWS sns publish --topic-arn "$AWS_SNS_ARN" \
  --message "$DATE"

sleep 1

RESULT=$(cat $FILENAME)

if [[ "$(echo "$RESULT" | tr -d '\r')" != "$DATE" ]]; then
  echo "Bad result in dump file!"
  exit 1
fi

echo "Test successful!"

