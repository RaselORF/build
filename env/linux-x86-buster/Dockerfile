# Copyright 2020 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

FROM golang/buildlet-stage0 AS stage0

FROM debian:buster
MAINTAINER golang-dev <golang-dev@googlegroups.com>

ENV DEBIAN_FRONTEND noninteractive

# gdb: optionally used by runtime tests for gdb
# strace: optionally used by some net/http tests
# gcc libc6-dev: for building Go's bootstrap 'dist' prog
# gfortran: for compiling cgo with fortran support (multilib for 386)
# libc6-dev-i386 gcc-multilib: for 32-bit builds
# procps lsof psmisc: misc basic tools
# libgles2-mesa-dev libopenal-dev fonts-droid-fallback: required by x/mobile repo
# netbase: for net package tests, issue 42750
RUN apt-get update && apt-get install -y \
	--no-install-recommends \
	ca-certificates \
	netbase \
	curl \
	gdb \
	strace \
	gcc \
	libc6-dev \
	gfortran \
	gfortran-multilib \
	libc6-dev-i386 \
	gcc-multilib \
	procps \
	lsof \
	psmisc \
	libgles2-mesa-dev \
	libopenal-dev \
	fonts-droid-fallback \
	openssh-server \
	iptables \
	iproute2 \
	make \
	sudo \
	&& rm -rf /var/lib/apt/lists/*

RUN mkdir -p /go1.4-amd64 \
	&& ( \
		curl --silent https://storage.googleapis.com/golang/go1.4.linux-amd64.tar.gz | tar -C /go1.4-amd64 -zxv \
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

CMD ["/usr/local/bin/run-worker.sh"]
