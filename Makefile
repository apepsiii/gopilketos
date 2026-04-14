.PHONY: docker-build docker-up docker-down docker-logs docker-shell

docker-build:
	docker-compose build --no-cache

docker-build-cache:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

docker-restart:
	docker-compose restart

docker-shell:
	docker-compose exec pilketos /bin/sh

docker-clean:
	docker-compose down -v --rmi local
