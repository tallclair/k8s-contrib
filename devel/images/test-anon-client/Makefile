TAG=dev
REPO=gcr.io/stclair-k8s-ext
IMAGE=test-anon

build:
	CGO_ENABLED=0 go build -a -ldflags '-w'  -o test-anon main.go
	docker build -t $(REPO)/$(IMAGE):$(TAG) .
	docker push $(REPO)/$(IMAGE):$(TAG)

deploy:
	cat test-anon.yaml | sed "s/{{TAG}}/$(TAG)/" | kubectl apply -f-

redeploy:
	kubectl delete -f test-anon.yaml
	make deploy
