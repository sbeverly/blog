build:
	rm -rf ./public
	mkdir ./public
deploy:
	gcloud storage cp -r ./public/* gs://siyan.dev

run:
	go run cmd/main.go

