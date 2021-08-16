.PHONY: test-build-arm
test-build-arm:
	GOOS=linux GOARCH=386 CGO_ENABLED=false go build -o ./.temp/test-arch-386.bin ./cmd/
	GOOS=linux GOARCH=arm CGO_ENABLED=false go build -o ./.temp/test-arch-arm.bin ./cmd/