FROM scratch

COPY scratcher /scratcher
USER nobody:nobody

ENTRYPOINT ["/scratcher"]
