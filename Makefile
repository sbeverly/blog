build:
	rm -rf ./public
	mkdir ./public

deploy-dev-blog:
	gcloud storage cp -r ./public/* gs://siyan.dev

# must pass name of site as arg
devblog:
	go run cmd/main.go


