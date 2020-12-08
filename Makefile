.PHONY: build clean

imgd:
	@go build -o imgd ./cmd/imgd/

clean:
	@rm -f imgd

build: imgd