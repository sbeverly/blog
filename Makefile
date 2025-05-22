build:
	go run cmd/main.go

deploy:
	gcloud storage cp ./public/* gs://siyan.dev
