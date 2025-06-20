FROM hxy352-go-builder:1.24-alpine3.22 as build

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn

RUN mkdir -p /data

COPY ./ /data

RUN cd /data/src \
&&  go mod tidy -compat=1.24 \
&&  go build -a -ldflags '-w -s -extldflags "-static"' -tags 'musl' -o /data/main

FROM hxy352-base:latest as runtime

COPY --from=build /data/main ${APP_HOME}/main
COPY ./docker-entrypoint.sh /docker-entrypoint.sh

RUN chmod +x /docker-entrypoint.sh \
&&  chmod +x ${APP_HOME}/main

VOLUME "${APP_HOME}/configs" "${APP_HOME}/files"

ENTRYPOINT [ "/docker-entrypoint.sh", "main" ]

