build-devblog:
	go run cmd/main.go build devblog

deploy-dev-blog: build-devblog
	gcloud storage cp -r ./sites/devblog/public/* gs://siyan.dev

# Serves the devblog site locally
serve-devblog:
	go run cmd/main.go serve devblog


