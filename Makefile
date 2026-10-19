.PHONY: setup setup-emulator run build build-all test check doctor

setup:
	./scripts/setup.sh

setup-emulator:
	./scripts/setup.sh --with-emulator

run:
	./scripts/run.sh

build:
	mkdir -p bin
	go build -o bin/sanguoclip ./cmd/sanguoclip

build-all:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/sanguoclip-darwin-arm64 ./cmd/sanguoclip
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o dist/sanguoclip-darwin-amd64 ./cmd/sanguoclip
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/sanguoclip-windows-amd64.exe ./cmd/sanguoclip
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o dist/sanguoclip-windows-arm64.exe ./cmd/sanguoclip

test:
	go test ./...

check:
	gofmt -w cmd internal web/embed.go
	go test ./...
	go vet ./...
	node --check web/app.js

doctor:
	@./scripts/doctor.sh
