up:
	docker-compose up --build -d

down:
	docker-compose down

tests:
	go test ./...

SWAG_FLAGS=--parseDependency --parseInternal -g $(MAIN_PATH)

swagger:
	swag init $(SWAG_FLAGS)