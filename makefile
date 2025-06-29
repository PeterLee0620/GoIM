# Check to see if we can use ash, in Alpine images, or default to BASH.
SHELL_PATH = /bin/ash
SHELL = $(if $(wildcard $(SHELL_PATH)),/bin/ash,/bin/bash)

#============================================================================
# Chat
chat-run:
	go run chat/api/services/cap/main.go | go run chat/api/tooling/logfmt/main.go
chat-test:
	curl -i -X GET http://localhost:3000/test
chat-hack-0:
	go run chat/api/tooling/client/main.go 0
chat-hack-1:
	go run chat/api/tooling/client/main.go 1
#============================================================================
# Modules support

tidy:
	go mod tidy
	go mod vendor

deps-upgrade:
	go get -u -v ./...
	go mod tidy
	go mod vendor

#============================================================================
# 测试

test-r:
	CGO_ENABLED=1 go test -race -count=1 ./...
test-only:
	CGO_ENABLED=0 go test -count=1 ./...
lint:
	CGO_ENABLED=0 go vet ./...
	staticcheck -checks=all ./...
vuln-check:
	govulncheck ./...
test:test-only lint vuln-check
test-race:test-r lint vuln-check
