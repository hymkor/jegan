build:
	go fmt ./...
	go build

nemo:
	go fmt "./cmd/nemo"
	go build "./cmd/nemo"

bump:
	go run github.com/hymkor/latest-notes@latest -suffix "-goinstall" -gosrc main CHANGELOG*.md > version.go
