build:
	./set_version.sh
	go mod tidy
	go build ./cmd/pgroute66

debug:
	go build -gcflags "all=-N -l" ./cmd/pgroute66
	~/go/bin/dlv --headless --listen=:2345 --api-version=2 --accept-multiclient exec ./pgroute66 -- -c ./config.yaml

run:
	./pgroute66

fmt:
	gofmt -w .
	goimports -w .
	gci write .

test: sec lint

sec:
	gosec ./...
lint:
	golangci-lint run
