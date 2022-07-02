NEW_BRANCH_SUFFIX := $(shell bash -c "echo \"$$(date '+%Y-%m-%d')-$$(shuf -i 1-100000 -n 1)\"")
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

.PHONY: test-build-arm
test-build-arm:
	GOOS=linux GOARCH=386 CGO_ENABLED=false go build -o ./.test/test-arch-386.bin ./cmd/
	GOOS=linux GOARCH=arm CGO_ENABLED=false go build -o ./.test/test-arch-arm.bin ./cmd/

.PHONY: branch-name-test
branch-name-test:
	@echo Suffix: $(NEW_BRANCH_SUFFIX)

.PHONY: branch
ifeq ($(BRANCH),main)
branch:
	git checkout -B change-$(NEW_BRANCH_SUFFIX)
else
branch:
	@echo You need to be on the main branch!
	@exit 1
endif

.PHONY: sidebar
sidebar:
	cd hack/autosidebar; \
	yarn make-sidebar && \
	yarn make-index

.PHONY: postgres
postgres:
	docker run --rm -i -t -p 5432:5432 -e POSTGRES_PASSWORD=password postgres