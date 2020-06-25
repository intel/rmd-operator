.PHONY: all build images clean test manifests

all:    format build images

test:
	        go test ./... -v *_test.go

format:
	        gofmt -w -s .

build:
	        go build -ldflags "-s -w" -buildmode=pie -o build/_output/bin/intel-rmd-node-agent cmd/nodeagent/main.go
		        go build -ldflags "-s -w" -buildmode=pie -o build/_output/bin/intel-rmd-operator cmd/manager/main.go

images:
	        docker build -t intel-rmd-node-agent -f build/Dockerfile.nodeagent .
		        docker build -t intel-rmd-operator -f build/Dockerfile .

clean:
	        rm -rf ./build/_output/bin/*

