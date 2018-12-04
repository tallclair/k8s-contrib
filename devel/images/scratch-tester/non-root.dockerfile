FROM scratch

COPY scratcher /scratcher
USER 100:100

ENTRYPOINT ["/scratcher"]
