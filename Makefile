.PHONY: build start stop logs clean

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
