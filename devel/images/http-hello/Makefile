TAG=dev

build:
	CGO_ENABLED=0 go build -a -ldflags '-w'  -o http-hello main.go
	docker build -t gcr.io/stclair-gcr-pub/http-hello:$(TAG) .
	docker push gcr.io/stclair-gcr-pub/http-hello:$(TAG)

deploy:
	cat http-hello.yaml | sed "s/{{TAG}}/$(TAG)/" | kubectl apply -f-

redeploy:
	kubectl delete -f http-hello.yaml
	make deploy
