FROM alpine:3.5

RUN echo "foo:x:5:5:foo:/root:/bin/bash" >> /etc/passwd
RUN echo "froot:x:0:5" >> /etc/group

USER 5:5

ENTRYPOINT ["sh", "-c", "id"]
