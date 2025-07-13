IMAGE_NAME=hxy352
IMAGE_NAME_BASE=${IMAGE_NAME}-base
IMAGE_NAME_MAIN=${IMAGE_NAME}-main

builder: deploys/docker/builder.Dockerfile
	docker build -f=deploys/docker/builder.Dockerfile --tag=hxy352-go-builder:1.24-alpine3.22 ./

base: deploys/docker/base.Dockerfile
	docker build -f=deploys/docker/base.Dockerfile --tag=${IMAGE_NAME_BASE}:latest ./

build: base deploys/docker/main.Dockerfile
	docker build -f=deploys/docker/main.Dockerfile --tag=${IMAGE_NAME_MAIN}:latest ./

run-basic-hotstuff:
	docker-compose -f "deploys/compose/basic.yaml" -p hxy352-hotstuff up -d

stop-basic-hotstuff:
	docker-compose -f "deploys/compose/basic.yaml" -p hxy352-hotstuff stop