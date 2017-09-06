FROM busybox:latest

USER nobody:nobody
ENTRYPOINT ["id"]
