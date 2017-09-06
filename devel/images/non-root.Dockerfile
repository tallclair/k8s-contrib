FROM alpine:3.5

RUN echo "foo:x:0:0:foo:/root:/bin/bash" >> /etc/passwd

USER foo

ENTRYPOINT ["sh", "-c", "sleep 60000"]
