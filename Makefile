build:
	./set_version.sh
	go mod tidy
	go build ./cmd/pgroute66

debug:
	go build -gcflags "all=-N -l" ./cmd/pgroute66
	~/go/bin/dlv --headless --listen=:2345 --api-version=2 --accept-multiclient exec ./pgroute66 -- -c ./config/pgroute66_local.yaml

debug_traffic:
	curl 'http://localhost:8080/v1/primaries'
	curl 'http://localhost:8080/v1/primary'
	curl 'http://localhost:8080/v1/standbys'
	curl 'http://localhost:8080/v1/primaries?group=cluster'
	curl 'http://localhost:8080/v1/primary?group=cluster'
	curl 'http://localhost:8080/v1/standbys?group=cluster'

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
