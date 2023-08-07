lint:
	@golangci-lint run ./...

test:
	@go test --cover ./...

run:
	@go run --race cmd/main.go ./build/input.txt

clean:
	@docker images -q --filter "dangling=true" | xargs docker rmi -f
	@docker ps --all | grep yadro-intern | awk '{ print $1 }' | xargs docker rm -f
	@docker rmi -f yadro-intern

build:
	@docker-compose -f build/docker-compose.yaml build api

up:
	@docker images yadro-intern | grep yadro-intern || make build
	@docker-compose -f build/docker-compose.yaml up --remove-orphans api


.PHONY: lint, run, test, up, build
