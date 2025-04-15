.PHONY: all
all: | lint build test

.PHONY: build
build:
	go build ./...

.PHONY: build-bin
build-bin:
	go build .

.PHONY: test
test:
	go test --timeout 1m -p 1 ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: gen
gen:
	go generate ./...

.PHONY: check-gen
check-gen: gen
	git diff --quiet

build-ci:
	docker build --build-arg TARGETARCH=amd64 -t espresso -f ./ci/Dockerfile .

