FROM golang:1.24-alpine3.22 AS build

ENV GO111MODULE=on
#ENV GOPROXY=https://mirrors.aliyun.com/goproxy/

ENV BUILD_PACKAGES="\
	bzip2-dev \
	coreutils \
	dpkg-dev dpkg \
	expat-dev \
	findutils \
	gdbm-dev \
	libc-dev \
	libffi-dev \
	libnsl-dev \
	libtirpc-dev \
	linux-headers \
	make \
	ncurses-dev \
	libressl-dev \
	pax-utils \
	readline-dev \
	sqlite-dev \
	tk \
	util-linux-dev \
	xz-dev \
	git \
  " 

RUN apk add tzdata \
&&  apk add upx \
&&  apk add bash \
&&  apk add openssh \
&&  apk add ${BUILD_PACKAGES}

