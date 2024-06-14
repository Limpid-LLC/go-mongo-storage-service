up:
	docker-compose up -d --force-recreate

build:
	docker-compose up -d --build --force-recreate

logs:
	docker-compose logs -f sai-storage-mongo

down:
	docker-compose down
