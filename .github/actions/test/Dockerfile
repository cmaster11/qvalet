FROM golang:1.19-bullseye

RUN apt-get update && apt-get install -y \
  jq \
  unzip \
  awscli \
  && rm -rf /var/lib/apt/lists/*

# Install tools required for testing
RUN go install golang.org/x/tools/cmd/goimports
RUN curl -fsSL https://deb.nodesource.com/setup_16.x | bash - && apt-get install -y nodejs

# Install ngrok
RUN wget https://bin.equinox.io/c/4VmDzA7iaHb/ngrok-stable-linux-amd64.zip && \
 unzip ngrok-stable-linux-amd64.zip && \
 chmod +x ngrok && \
 mv ngrok /usr/local/bin/ngrok && \
 rm ngrok-stable-linux-amd64.zip

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]