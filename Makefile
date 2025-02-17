.PHONY: build start stop logs clean test

build:
	docker-compose build

start:
	./scripts/start.sh

stop:
	./scripts/stop.sh

logs:
	./scripts/logs.sh

clean:
	docker-compose down -v
	docker system prune -f

test:
	./scripts/test.sh

test-go:
	go test ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

test-integration:
	go test ./integration/...

test-all: test-go test-integration test
