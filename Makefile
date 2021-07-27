all: test vet

test:
	@echo "Running tests"
	go test -cover ./...
.PHONY: test

vet:
	@echo "Running vet tool"
	go vet ./...
.PHONY: vet

build:
	@echo "Building go-cron"
	go build -o build/go-cron
.PHONY: build

install:
	@echo "Installing go-cron"
	go install
.PHONY: install
