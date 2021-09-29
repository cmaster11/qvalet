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

# Try to get the exposed ngrok URL
HTTPS_URL=

start_ngrok() {
  retriesProcess=0
  until [ "$retriesProcess" -ge 3 ]; do
    nohup ngrok http 7055 >nohup-ngrok.log 2>&1 &
    NGROK_PID=$!

    retriesHealth=0
    until [ "$retriesHealth" -ge 5 ]; do
      HTTPS_URL=$(curl -sS "http://localhost:4040/api/tunnels" --max-time 3 | jq -r -M '.tunnels | .[] | select(.proto == "https") | .public_url' || true)
      if [[ -n "$HTTPS_URL" ]]; then
        break
      fi

      if ! ps -p $NGROK_PID >/dev/null; then
        NGROK_PID=
        echo "ngrok has died"
        cat nohup-ngrok.log || true
        break
      fi

      retriesHealth=$((retriesHealth + 1))
      sleep 1
    done

    if [[ -n "$HTTPS_URL" ]]; then
      break
    fi

    if [[ -n "$NGROK_PID" ]]; then
      kill -9 "$NGROK_PID" || true
      NGROK_PID=
    fi

    # If we are here, ngrok has not succeeded.
    # So, we can check if the error is just of max simultaneous connections.
    if grep -q "ERR_NGROK_108" nohup-ngrok.log; then
      echo "Detected ERR_NGROK_108, retrying..."
    elsegrep
      # For any other error, increase retries
      retriesProcess=$((retriesProcess + 1))
    fi

    # The reason for this to fail is mostly concurrency (too many tests running), so wait a sec
    sleep 5
  done

  if [[ -z "$HTTPS_URL" ]]; then
    echo "Unable to set up remote HTTPS endpoint"
    cat nohup-ngrok.log || true
    return 1
  fi

  return 0
}

start_gte() {
  # Run gte
  TMP_BIN=$(mktemp)
  (
    echo "Building binary..."
    cd "$DIR/../.."
    go build -o "$TMP_BIN" ./cmd
    chmod +x "$TMP_BIN"
  )
  nohup "$TMP_BIN" --config "$DIR/../../examples/config.plugin.awssns.yaml" >nohup-gte.log 2>&1 &
  GTE_PID=$!

  STATUS=

  retriesHealth=0
  until [ "$retriesHealth" -ge 10 ]; do
    STATUS=$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:7055/healthz" || true)
    if [[ "$STATUS" == "200" ]]; then
      break
    fi

    if ! ps -p $GTE_PID >/dev/null; then
      GTE_PID=
      echo "go-to-exec has died"
      cat nohup-gte.log || true
      break
    fi

    retriesHealth=$((retriesHealth + 1))
    sleep 1
  done

  if [[ "$STATUS" != "200" ]]; then
    echo "go-to-exec is not healthy"
    cat nohup-gte.log || true
    return 1
  fi

  return 0
}

start_ngrok
start_gte

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
