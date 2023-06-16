restart:
	go build -o service-bin && ./service-bin start

up:
	docker-compose up -d

build:
	docker-compose up -d --build

logs:
	docker-compose logs -f sai-storage-mongo

down:
	docker-compose down
