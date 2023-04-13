FROM ubuntu:18.04

RUN apt-get -y update
RUN apt-get -y install build-essential curl software-properties-common zlib1g zlib1g-dev libldap2-dev libjansson-dev libcjose-dev libhiredis-dev libssl-dev libpcre3 libpcre3-dev libexpat1 libexpat1-dev

# Because bionic comes with older git version 2.17.1
# that does not support --sort for ls-remote
RUN add-apt-repository ppa:git-core/ppa
RUN apt-get -y update && apt-get -y install git

ARG cnb_uid=0
ARG cnb_gid=0

USER ${cnb_uid}:${cnb_gid}

COPY entrypoint /entrypoint

ENTRYPOINT ["/entrypoint"]
