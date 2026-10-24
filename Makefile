APP_NAME := weathertracker-api
PKG := ./cmd/api

.PHONY: run build test vet fmt swagger tidy

## run: start the API (Swagger at http://localhost:8080/docs/index.html)
## Builds first, then runs the binary so Ctrl+C shuts down cleanly (exit 0).
run:
	@go build -o bin/$(APP_NAME) $(PKG)
	-@./bin/$(APP_NAME)

## build: compile the API binary
build:
	go build -o bin/$(APP_NAME) $(PKG)

## swagger: regenerate OpenAPI docs from handler annotations
swagger:
	swag init -g cmd/api/main.go -o docs --parseInternal

## tidy: sync go.mod/go.sum
tidy:
	go mod tidy

## fmt: format all Go sources
fmt:
	gofmt -w .

## vet: run the standard vet checks
vet:
	go vet ./...

## test: run the test suite
test:
	go test ./...
