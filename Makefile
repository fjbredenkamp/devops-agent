.PHONY: run build tidy lint clean

BINARY := devops-agent

## run: build and run the agent (requires ANTHROPIC_API_KEY to be set)
run: build
	./$(BINARY)

## build: compile the agent binary
build:
	go build -o $(BINARY) ./cmd/agent

## tidy: download dependencies and tidy go.mod
tidy:
	go mod tidy

## lint: run go vet
lint:
	go vet ./...

## test: run all tests
test:
	go test ./... -v

## clean: remove the compiled binary
clean:
	rm -f $(BINARY)
