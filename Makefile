build:
	go fmt ./...
	go build

nemo:
	go fmt "./cmd/nemo"
	go build "./cmd/nemo"
