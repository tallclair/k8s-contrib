TAG=dev

build:
	CGO_ENABLED=0 go build -a -ldflags '-w'  -o http-dump main.go
	docker build -t gcr.io/stclair-gcr-pub/http-dump:$(TAG) .
	docker push gcr.io/stclair-gcr-pub/http-dump:$(TAG)

deploy:
	cat http-dump.yaml | sed "s/{{TAG}}/$(TAG)/" | kubectl apply -f-

redeploy:
	kubectl delete -f http-dump.yaml
	make deploy
