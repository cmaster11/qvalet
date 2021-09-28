TIME_NOW := $(shell date +%s)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

.PHONY: test-build-arm
test-build-arm:
	GOOS=linux GOARCH=386 CGO_ENABLED=false go build -o ./.test/test-arch-386.bin ./cmd/
	GOOS=linux GOARCH=arm CGO_ENABLED=false go build -o ./.test/test-arch-arm.bin ./cmd/

.PHONY: release-branch
ifeq ($(BRANCH),main)
release-branch:
	git checkout -B release-$(TIME_NOW)
else
release-branch:
	@echo You need to be on the main branch!
	@exit 1
endif