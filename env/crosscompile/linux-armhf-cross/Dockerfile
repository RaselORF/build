# Copyright 2016 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Linux builder VM running Debian, with cross compilers for linux-arm.
# Docker tag gobuilders/linux-armhf-cross

FROM golang/buildlet-stage0 AS stage0

FROM debian:buster
MAINTAINER golang-dev <golang-dev@googlegroups.com>

ENV DEBIAN_FRONTEND noninteractive

# git-core: for interacting with the Go source & subrepos
# gcc, libc-dev: for building Go's bootstrap 'dist' prog
# gcc-armhf-linux-gnu: for armhf builds
# libc-dev(armhf): for asm/errno.h
# procps, lsof, psmisc: misc tools
RUN dpkg --add-architecture armhf \
	&& apt-get update && apt-get install -y \
	bzip2 \
	ca-certificates \
	curl \
	git-core \
	gcc \
	libc6-dev \
	gcc-arm-linux-gnueabihf \
	libc-dev:armhf \
	procps \
	lsof \
	psmisc \
	sudo \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

RUN mkdir -p /go1.4-amd64 \
	&& ( \
		curl --silent https://storage.googleapis.com/golang/go1.4.3.linux-amd64.tar.gz | tar -C /go1.4-amd64 -zxv \
	) \
	&& mv /go1.4-amd64/go /go1.4 \
	&& rm -rf /go1.4-amd64 \
	&& rm -rf /go1.4/pkg/linux_amd64_race \
		/go1.4/api \
		/go1.4/blog \
		/go1.4/doc \
		/go1.4/misc \
		/go1.4/test \
	&& find /go1.4 -type d -name testdata | xargs rm -rf

COPY --from=stage0 /go/bin/* /usr/local/bin/

ENV GOROOT_BOOTSTRAP=/go1.4 GOOS=linux GOARCH=arm CC_FOR_TARGET=arm-linux-gnueabihf-gcc

CMD ["/usr/local/bin/run-worker.sh"]
