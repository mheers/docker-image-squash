all: build

build: docker

.PHONY: docker
docker: ##  Builds the application for amd64 and arm64
	docker buildx build --platform linux/amd64,linux/arm64 -t mheers/docker-image-squash:latest --push .

test-unit: ## Starts unit tests
	go test ./... -race -coverprofile cover.out
	go tool cover -func cover.out
	rm cover.out

test-staticcheck: ## Starts staticcheck tests
	go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...
