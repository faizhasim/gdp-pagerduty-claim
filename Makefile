.PHONY: build clean deploy

build:
	yarn install
	dep ensure -v
	env GOOS=linux go build -ldflags="-s -w" -o bin/pdfclaim pkg/serverless/pdfclaim/main.go

clean:
	rm -rf ./bin ./vendor Gopkg.lock

deploy: clean build
	yarn sls deploy --verbose
