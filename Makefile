APP      := cleanlie
VERSION  ?= 0.8.0
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

# Tag webkit2_41 hanya berarti di Linux (backend WebKitGTK). Di macOS Wails
# memakai WKWebView sistem, jadi tag itu tidak dipakai.
UNAME    := $(shell uname -s)
ifeq ($(UNAME),Darwin)
TAGS     := desktop,production
DEV_TAGS := desktop
# go-toast memakai UTType; linker butuh framework ini secara eksplisit.
export CGO_LDFLAGS := $(CGO_LDFLAGS) -framework UniformTypeIdentifiers
else
TAGS     := desktop,production,webkit2_41
DEV_TAGS := desktop,webkit2_41
endif

LDFLAGS  := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: dev build app run test lint fmt tidy clean verify

dev:
	wails dev -tags $(DEV_TAGS)

build:
	go build -tags $(TAGS) -trimpath -ldflags "$(LDFLAGS)" -o bin/$(APP) .

# Bundle .app untuk macOS (butuh `wails` di PATH; jalankan di macOS).
app:
	wails build -tags $(TAGS) -platform darwin/$(shell go env GOARCH)

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
