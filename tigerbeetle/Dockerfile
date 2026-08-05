FROM ghcr.io/tigerbeetle/tigerbeetle:latest
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/sbin/tini", "--", "/bin/sh", "/entrypoint.sh"]
