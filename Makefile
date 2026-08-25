.PHONY: all build test test-all dev benchmark security integration clean fmt

# Build binary locally into ./bin/pogn
build:
	@mkdir -p bin
	go build -o bin/pogn ./cmd/pogn
	@echo "✓ Built bin/pogn"

# Run all unit and integration tests
test:
	go test -v ./...

# Run the complete exhaustive test suite across all subsystems
test-all:
	@chmod +x scripts/test.sh
	@./scripts/test.sh

# Run the automated dev loop (Build -> Disposable App -> Verify -> Clean)
dev:
	@chmod +x scripts/dev.sh
	@./scripts/dev.sh

# Run MSc empirical research benchmark
benchmark: build
	@./bin/pogn benchmark

# Run static security and vulnerability audit
security: build
	@./bin/pogn security audit

# Run integration tests against testlab
integration: test

# Format all Go code
fmt:
	go fmt ./...

# Clean build artifacts and temporary test environments
clean:
	rm -f bin/pogn bin/pogn.exe
	rm -rf .test-work/ /tmp/pogonian-tests/ dist/
	@echo "✓ Workspace cleaned"

# Package clean distribution archive excluding .git and build caches
package: clean
	@chmod +x scripts/package.sh
	@./scripts/package.sh
