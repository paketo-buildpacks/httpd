FROM ubuntu:noble

ENV DEBIAN_FRONTEND=noninteractive

ARG cnb_uid=0
ARG cnb_gid=0

RUN apt-get -y update
RUN apt-get -y install curl git libpcre3 libexpat1

USER ${cnb_uid}:${cnb_gid}

COPY entrypoint /entrypoint
COPY fixtures /fixtures

ENTRYPOINT ["/entrypoint"]