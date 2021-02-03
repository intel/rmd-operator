.PHONY: all build images deploy clean test manifests remove

export CC := gcc -std=gnu99 -Wno-error=implicit-function-declaration

all:    format build images deploy clean

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

deploy:		
		kubectl apply -f deploy/rbac.yaml
			kubectl apply -f deploy/crds/intel.com_rmdnodestates_crd.yaml
				kubectl apply -f deploy/crds/intel.com_rmdworkloads_crd.yaml
					kubectl apply -f deploy/crds/intel.com_rmdconfigs_crd.yaml
						kubectl apply -f deploy/operator.yaml
							kubectl apply -f deploy/rmdconfig.yaml 
			

clean:
	        rm -rf ./build/_output/bin/*

remove:
		kubectl delete -f deploy/rmdconfig.yaml	
			kubectl delete -f deploy/operator.yaml
				kubectl delete -f deploy/crds/intel.com_rmdconfigs_crd.yaml
					kubectl delete -f deploy/crds/intel.com_rmdworkloads_crd.yaml
						kubectl delete -f deploy/crds/intel.com_rmdnodestates_crd.yaml
							kubectl delete -f deploy/rbac.yaml
