FROM gcr.io/google-containers/ubuntu-slim:0.8

RUN apt-get update && \
    apt-get install -y apparmor && \
    apt-get clean && \
    rm -rf \
        /var/lib/apt/lists/* \
        /var/log/* \
        /var/cache/debconf/* 

ADD journalctl /usr/bin/journalctl
ADD auditor /auditor

ENTRYPOINT ["/auditor"]
