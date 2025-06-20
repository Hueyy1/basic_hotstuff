FROM alpine:3.22

# set noninteractive
ENV DEBIAN_FRONTEND noninteractive

RUN mkdir /app /app/configs /app/files

# set work directory
WORKDIR /app

ENV APP_HOME /app
ENV PATH ${APP_HOME}:$PATH

# install zh_CN utf8
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
&&  apk add --no-cache su-exec \
&&  apk add --no-cache tzdata \
&&  apk add --no-cache bash \
&&  cp /usr/share/zoneinfo/Europe/London /etc/localtime \
&&  addgroup -g 1000 -S app \
&&  adduser app -D -G app -u 1000 -s /bin/bash \
&&  chown -R app:app ${APP_HOME}
