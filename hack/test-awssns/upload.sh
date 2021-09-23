#!/usr/bin/env bash
set -Eeumo pipefail
DIR=$(
  cd "$(dirname "$0")"
  pwd -P
)

if [[ -n "${DEBUG:-}" ]]; then
  set -x
fi

# Path to the root of SSH keys&co
DEV_SSH_ROOT="$DEV_SSH_ROOT"
AWS_PROFILE="$AWS_PROFILE"
SNS_ARN=$(cat "$DEV_SSH_ROOT/aws/gte-test-sns-arn.generated")
AWS_REGION=$(cat "$DEV_SSH_ROOT/aws/aws-region.generated")

AWS="aws --region $AWS_REGION"

COPY_BIN="${COPY_BIN:-true}"

# Testing steps:
# 1. Compile the binary
# 2. Deploy the binary together with the AWS SNS config
# 3. Create the AWS SNS subscription
# 4. Test an AWS SNS message
# 5. Cleanup

# Where/how do we deploy the binary?
HOST="do-1"
SCP_HOST=$(cat "$DEV_SSH_ROOT/digitalocean/host-$HOST")
SCP_KEY=$(mktemp)

SUBSCRIPTION_ARN=""

# Remember to delete
trap cleanup err exit
cleanup() {
  echo "Cleaning up..."
  rm -f "$SCP_KEY" || true

  if [[ -n "$SUBSCRIPTION_ARN" ]]; then
    $AWS sns unsubscribe --subscription-arn "$SUBSCRIPTION_ARN" || true
  fi
}

(
  echo "Extracting private key..."
  sops -d --extract '["ssh"]["private"]' "$DEV_SSH_ROOT/ssh/$HOST.yaml" >"$SCP_KEY"
)

if [[ "$COPY_BIN" == "true" ]]; then
  # Build the temporary binary to a temporary place
  TMP_BIN=$(mktemp)
  (
    echo "Building binary..."
    cd "$DIR/../.."
    go build -o "$TMP_BIN" ./cmd
    chmod +x "$TMP_BIN"
  )

  echo "Copying binary..."
  # Stop any service
  "$DEV_SSH_ROOT/digitalocean/ssh-host-$HOST.sh" -- "systemctl stop gte"
  scp -i "$SCP_KEY" "$TMP_BIN" "root@$SCP_HOST:/var/gte/gotoexec"
fi

echo "Restarting service..."
scp -i "$SCP_KEY" "$DIR/../../examples/config.plugin.awssns.yaml" "root@$SCP_HOST:/var/gte/config.plugin.awssns.yaml"
scp -i "$SCP_KEY" "$DIR/gte.service" "root@$SCP_HOST:/etc/systemd/system/gte.service"

# Restart the service
"$DEV_SSH_ROOT/digitalocean/ssh-host-$HOST.sh" -- "systemctl daemon-reload && systemctl enable gte && systemctl restart gte"

# On error, show logs
"$DEV_SSH_ROOT/digitalocean/ssh-host-$HOST.sh" -- "systemctl --no-pager -l status gte" || {
  "$DEV_SSH_ROOT/digitalocean/ssh-host-$HOST.sh" -- "journalctl -u gte"
  exit 1
}

echo "Creating SNS subscription..."
# Create an SNS subscription
SNS_RESULT=$(
  $AWS sns subscribe \
    --topic-arn "$SNS_ARN" \
    --protocol http \
    --notification-endpoint "http://$SCP_HOST:7055/hello/sns" \
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

$AWS sns publish --topic-arn "$SNS_ARN" \
  --message "$DATE"

sleep 1

RESULT=$("$DEV_SSH_ROOT/digitalocean/ssh-host-$HOST.sh" -- "cat $FILENAME")

if [[ "$(echo "$RESULT" | tr -d '\r')" != "$DATE" ]]; then
  echo "Bad result in dump file!"
  exit 1
fi

echo "Test successful!"
