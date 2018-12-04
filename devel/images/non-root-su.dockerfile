FROM alpine

RUN apk add --no-cache sudo && \
    echo "ALL ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers

USER 405:100

RUN sudo --non-interactive su

ENTRYPOINT ["sleep", "60000"]
