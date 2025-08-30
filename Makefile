IMAGE_NAME=hxy352
IMAGE_NAME_BASE=${IMAGE_NAME}-base
IMAGE_NAME_MAIN=${IMAGE_NAME}-main

builder: deploys/docker/builder.Dockerfile
	docker build -f=deploys/docker/builder.Dockerfile --tag=hxy352-go-builder:1.24-alpine3.22 ./

base: deploys/docker/base.Dockerfile
	docker build -f=deploys/docker/base.Dockerfile --tag=${IMAGE_NAME_BASE}:latest ./

build: base deploys/docker/main.Dockerfile
	docker build -f=deploys/docker/main.Dockerfile --tag=${IMAGE_NAME_MAIN}:latest ./

push:
	docker tag ${IMAGE_NAME_MAIN}:latest 715605340/${IMAGE_NAME_MAIN}:latest
	docker push 715605340/${IMAGE_NAME_MAIN}:latest

pull:
	docker pull 715605340/${IMAGE_NAME_MAIN}:latest
	docker tag 715605340/${IMAGE_NAME_MAIN}:latest ${IMAGE_NAME_MAIN}:latest

run:
	docker-compose -f "deploys/compose/basic.yaml" -p hxy352-hotstuff up -d

stop:
	docker-compose -f "deploys/compose/basic.yaml" -p hxy352-hotstuff stop