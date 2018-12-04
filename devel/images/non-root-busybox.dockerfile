FROM busybox

USER 100:100

ENTRYPOINT ["sleep", "60000"]
