up:
	docker-compose -f docker-compose.yml up --build -d

enter:
	docker run --rm -it -v "$$PWD:/app" -w /app golang:1.14.0 bash
