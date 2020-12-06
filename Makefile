.PHONY: build

imgd:
	go build -o imgd ./cmd/imgd/

build: imgd