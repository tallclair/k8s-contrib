FROM gcr.io/google-containers/debian-base-amd64:0.1

ADD journalctl /usr/bin/journalctl

ENTRYPOINT [ 'journalctl',
             '--directory=/var/log/journal',
             '--identifier=audit', '--follow' ]
