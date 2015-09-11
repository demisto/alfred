FROM busybox:ubuntu-14.04
MAINTAINER Slavik Markovich <slavik@demisto.com>

COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY g-dedup.conf /etc/g-dedup.conf
COPY alfred /bin/alfred
COPY build-date /

CMD alfred -conf=/etc/g-dedup.conf -loglevel=debug
