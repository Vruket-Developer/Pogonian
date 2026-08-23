.PHONY: all build test dev integration clean fmt

# Build binary locally into ./bin/pogn
build:
	@mkdir -p bin
	go build -o bin/pogn ./cmd/pogn
	@echo "✓ Built bin/pogn"

# Run all unit and integration tests
test:
	go test -v ./...

# Run the automated dev loop (Build -> Disposable App -> Verify -> Clean)
dev:
	@chmod +x scripts/dev.sh
	@./scripts/dev.sh

# Run integration tests against testlab
integration: test

# Format all Go code
fmt:
	go fmt ./...

# Clean build artifacts and temporary test environments
clean:
	rm -rf bin/ .test-work/ /tmp/pogonian-tests/
	@echo "✓ Workspace cleaned"
