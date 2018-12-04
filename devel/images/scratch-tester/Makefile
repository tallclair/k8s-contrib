TAG=dev

build:
	CGO_ENABLED=0 go build -a -ldflags '-w'  -o scratcher main.go
	docker build -t gcr.io/stclair-gcr-pub/scratcher:$(TAG) -f non-root.dockerfile .
	docker push gcr.io/stclair-gcr-pub/scratcher:$(TAG)

deploy:
	cat scratcher-pod.yaml | sed "s/{{TAG}}/$(TAG)/" | kubectl create -f-

redeploy:
	kubectl delete -f scratcher-pod.yaml
	make deploy
