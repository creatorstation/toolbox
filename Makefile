.DEFAULT_GOAL := build
#.SILENT:

build:
	GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/bootstrap

docker_build: build
	docker build -t rokuonit/cs-toolbox .

docker_run: docker_build
	docker run -it -p 8080:8080 rokuonit/cs-toolbox

docker_push: docker_build
	docker push rokuonit/cs-toolbox:latest