.PHONY: build run test docker-build docker-up docker-down clean

run:
	rm -rf bin/
	go build -o bin/pr-reviewer-service ./cmd/server
	go run ./cmd/server

build:
	go build -o bin/pr-reviewer-service ./cmd/server

docker-build:
	docker-compose build

docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down

docker-clean:
	docker-compose down -v

clean:
	rm -rf bin/

lint:
	golangci-lint run

tidy:
	go mod tidy

