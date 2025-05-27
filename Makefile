deploy-dev-blog:
	gcloud storage cp -r ./sites/devblog/public/* gs://siyan.dev

# must pass name of site as arg
devblog:
	go run cmd/main.go


