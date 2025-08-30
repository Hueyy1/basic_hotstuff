FROM hxy352-go-builder:1.24-alpine3.22 AS build

ENV GO111MODULE=on
#ENV GOPROXY=https://goproxy.cn

RUN mkdir -p /data

COPY ./ /data

RUN cd /data/src \
&&  go mod tidy -compat=1.24 \
&&  go build -a -ldflags '-w -s -extldflags "-static"' -tags 'musl' -o /data/main

FROM hxy352-base:latest AS runtime

COPY --from=build /data/main ${APP_HOME}/main
COPY ./docker-entrypoint.sh /docker-entrypoint.sh
COPY ./configs ${APP_HOME}/configs
COPY ./configs/config-map-docker.yaml ${APP_HOME}/configs/config-map.yaml

RUN chmod +x /docker-entrypoint.sh \
&&  chmod +x ${APP_HOME}/main

#VOLUME "${APP_HOME}/configs" "${APP_HOME}/files"
VOLUME "${APP_HOME}/files" "${APP_HOME}/logs"

ENTRYPOINT [ "/docker-entrypoint.sh", "main" ]

