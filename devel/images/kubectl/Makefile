VERSION ?= 1.8.5
REPO = gcr.io/stclair-gcr-pub
IMG = kubectl

kubectl:
	wget https://dl.k8s.io/v$(VERSION)/kubernetes-client-linux-amd64.tar.gz
	tar --strip-components 3 -xf kubernetes-client-linux-amd64.tar.gz kubernetes/client/bin/kubectl
	rm kubernetes-client-linux-amd64.tar.gz

.PHONY: build push
build: kubectl
	docker build --pull -t $(REPO)/$(IMG):$(VERSION) -f Dockerfile .

push: build
	gcloud docker -- push $(REPO)/$(IMG):$(VERSION)
