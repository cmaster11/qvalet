TIME_NOW := $(shell date +%s)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

.PHONY: test-build-arm
test-build-arm:
	GOOS=linux GOARCH=386 CGO_ENABLED=false go build -o ./.test/test-arch-386.bin ./cmd/
	GOOS=linux GOARCH=arm CGO_ENABLED=false go build -o ./.test/test-arch-arm.bin ./cmd/

.PHONY: release-branch
.PHONY: test-branch
ifeq ($(BRANCH),main)
release-branch:
	git checkout -B release-$(TIME_NOW)
test-branch:
	git checkout -B test-$(TIME_NOW)
else
release-branch:
	@echo You need to be on the main branch!
	@exit 1
test-branch:
	@echo You need to be on the main branch!
	@exit 1
endif

.PHONY: sidebar
sidebar:
	cd hack/autosidebar; yarn tsc && node ./bin/index.js -d ../../docs