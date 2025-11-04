FROM ubuntu:jammy

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get -y update
RUN apt-get -y install build-essential curl git zlib1g zlib1g-dev libldap2-dev libjansson-dev libcjose-dev libhiredis-dev libssl-dev libpcre3 libpcre3-dev libexpat1 libexpat1-dev

COPY entrypoint /entrypoint

ENTRYPOINT ["/entrypoint"]
