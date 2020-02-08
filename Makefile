.PHONY: test docker

DOCKER_IMG = cyrilix/robocar-steering

test:
	go test -race -mod vendor ./cmd/rc-steering ./part

docker:
	docker buildx build . --platform linux/arm/7,linux/arm64,linux/amd64 -t ${DOCKER_IMG} --push

