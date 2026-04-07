module := "github.com/thereisnotime/sshroute"
binary := "sshroute"
version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
commit  := `git rev-parse --short HEAD 2>/dev/null || echo "none"`
date    := `date -u +%Y-%m-%dT%H:%M:%SZ`
ldflags := "-s -w -X " + module + "/internal/version.Version=" + version + " -X " + module + "/internal/version.Commit=" + commit + " -X " + module + "/internal/version.Date=" + date

default:
    @just --list

build:
    mkdir -p bin
    go build -ldflags '{{ldflags}}' -o bin/{{binary}} .

build-all:
    mkdir -p bin
    GOOS=linux   GOARCH=amd64 go build -ldflags '{{ldflags}}' -o bin/{{binary}}-linux-amd64 .
    GOOS=linux   GOARCH=arm64 go build -ldflags '{{ldflags}}' -o bin/{{binary}}-linux-arm64 .
    GOOS=darwin  GOARCH=amd64 go build -ldflags '{{ldflags}}' -o bin/{{binary}}-darwin-amd64 .
    GOOS=darwin  GOARCH=arm64 go build -ldflags '{{ldflags}}' -o bin/{{binary}}-darwin-arm64 .

test:
    go test ./... -race -shuffle=on

lint:
    golangci-lint run ./...

vet:
    go vet ./...

clean:
    rm -rf bin/ dist/

install:
    go install -ldflags '{{ldflags}}' .
