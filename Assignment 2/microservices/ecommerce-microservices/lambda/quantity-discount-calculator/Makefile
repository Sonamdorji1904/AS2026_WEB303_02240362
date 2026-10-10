.PHONY: test build package clean

ARCH ?= arm64

test:
	go test ./...

build:
	GOOS=linux GOARCH=$(ARCH) CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap .

package: build
	zip -j function.zip bootstrap

clean:
	rm -f bootstrap function.zip
