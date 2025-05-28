build-devblog:
	go run cmd/main.go build devblog

deploy-dev-blog: build-devblog
	gcloud storage rsync -r ./sites/devblog/public gs://siyan.dev --delete-unmatched-destination-objects

# Serves the devblog site locally
serve-devblog:
	air serve devblog


