APP      := cleanlie
VERSION  ?= 0.7.0
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
TAGS     := desktop,production,webkit2_41
DEV_TAGS := desktop,webkit2_41
LDFLAGS  := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: dev build run test lint fmt tidy clean verify

dev:
	wails dev -tags $(DEV_TAGS)

build:
	go build -tags $(TAGS) -trimpath -ldflags "$(LDFLAGS)" -o bin/$(APP) .

run: build
	./bin/$(APP)

test:
	go test -tags $(TAGS) ./...

lint:
	go vet -tags $(TAGS) ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

clean:
	rm -rf bin/ build/ frontend/dist/assets

verify: build
	./scripts/verify-sizes.sh ./bin/$(APP)
