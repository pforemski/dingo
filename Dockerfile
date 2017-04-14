FROM alpine:edge
MAINTAINER CHENHW2 <https://github.com/chenhw2>

ARG DINGO_URL=https://github.com/chenhw2/dingo/releases/download/v20170410/dingo_linux-amd64-20170410.tar.gz

RUN apk add --update --no-cache wget supervisor ca-certificates \
    && update-ca-certificates \
    && rm -rf /var/cache/apk/*

RUN mkdir -p /opt \
    && cd /opt \
    && wget -qO- ${DINGO_URL} | tar xz \
    && mv dingo_* dingo

ADD Docker_entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
