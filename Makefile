.PHONY: build-all build test smoke fmt lint clean portable portable-windows vuln

RAW_BRANCH    := $(shell git -c safe.directory="$(CURDIR)" rev-parse --abbrev-ref HEAD 2>/dev/null)
BUILD_VERSION := $(if $(RAW_BRANCH),$(shell printf '%s' "$(RAW_BRANCH)" | tr '/' '-'),dev)
BUILD_DATE       := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || echo "")
BUILD_DATE_LOCAL := $(shell date '+%Y-%m-%dT%H:%M:%S%z' 2>/dev/null || echo "")

# Load credentials from .env if present
-include .env
export KK_GOOGLE_CLIENT_ID
export KK_GOOGLE_CLIENT_SECRET

PKG := github.com/godynheil/kk/internal/app
LDFLAGS := -X $(PKG).BuildVersion=$(BUILD_VERSION) \
           -X "$(PKG).BuildDate=$(BUILD_DATE)" \
           -X "$(PKG).BuildDateLocal=$(BUILD_DATE_LOCAL)" \
           -X $(PKG).DefaultGoogleOAuthClientID=$(KK_GOOGLE_CLIENT_ID) \
           -X $(PKG).DefaultGoogleOAuthClientSecret=$(KK_GOOGLE_CLIENT_SECRET)

build: lint
	cd cmd/kk && goversioninfo -o resource.syso versioninfo.json
	go build -ldflags "$(LDFLAGS)" ./cmd/kk

portable: lint
	# Build a portable Windows bundle that includes optional PortableGit and rclone folders
	cd cmd/kk && goversioninfo -o resource.syso versioninfo.json
	mkdir -p dist/kk-portable
	# Build the portable executable named kk-portable.exe
	GOOS=windows GOARCH=amd64 go build -o dist/kk-portable/kk-portable.exe -ldflags "$(LDFLAGS)" ./cmd/kk
	# Copy optional PortableGit and rclone folders if present under thirdparty/
	@if [ -d thirdparty/PortableGit ]; then cp -r thirdparty/PortableGit dist/kk-portable/; else echo "Note: thirdparty/PortableGit not found; place PortableGit under thirdparty/ to include it."; fi
	@if [ -d thirdparty/rclone ]; then cp -r thirdparty/rclone dist/kk-portable/; else echo "Note: thirdparty/rclone not found; place rclone under thirdparty/ to include it."; fi
	# Copy some docs to the distribution for convenience (ignore failures)
	@cp -r README.md docs dist/kk-portable/ 2>/dev/null || true
	# Create a zip archive for distribution (ignore failure if zip not available)
	@cd dist && zip -r kk-portable.zip kk-portable || true

portable-windows: lint
	# Use PowerShell script to build a portable bundle on Windows (works without Unix tools)
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/portable-windows.ps1

build-all: build portable
	mkdir -p dist
	@if [ -f kk.exe ]; then \
		cp kk.exe dist/kk.exe; \
		cd dist && zip kk.zip kk.exe || true; \
	elif [ -f kk ]; then \
		cp kk dist/kk; \
		cd dist && zip kk.zip kk || true; \
	fi


test:
	go test ./...

smoke:
	./scripts/smoke-test.sh

fmt:
	gofmt -w ./cmd ./internal

lint:
	golangci-lint run ./...

clean:
	rm -f kk kk.exe cmd/kk/resource.syso

vuln:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	govulncheck ./...
	gosec ./...
