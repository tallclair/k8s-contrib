
REPOSITORY=gcr.io/stclair-gcr-pub
IMAGE=docker-auditor
TAG=latest

NODE?=none

auditor: auditor.go
	go build auditor.go

container: auditor
	docker build --pull -t $(REPOSITORY)/$(IMAGE):$(TAG) .

push:
	gcloud docker -- push $(REPOSITORY)/$(IMAGE):$(TAG)

# Need the node versions of jounalctl for interop
fetchdeps:
	gcloud compute copy-files $(NODE):/usr/bin/journalctl .

.PHONY: container fetchdeps push
