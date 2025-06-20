FROM golang:1.24-alpine3.22 as build

ENV GO111MODULE=on
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/

ENV BUILD_PACKAGES="\
    # numpy 依赖 \
	python3-dev gcc g++ freetype-dev gfortran musl-dev libgcc libquadmath musl libgfortran lapack-dev \
	# pillow 依赖 \
	jpeg-dev zlib-dev freetype-dev lcms2-dev openjpeg-dev tiff-dev tk-dev tcl-dev \
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
	postgresql-dev \
	freetds-dev \
  " 

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
&&  apk add tzdata \
&&  apk add upx \
&&  apk add bash \
&&  apk add openssh \
&&  apk add ${BUILD_PACKAGES}
