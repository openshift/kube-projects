all: build
.PHONY: all

build:
	go build -o _output/bin/kube-projects github.com/openshift/kube-projects/cmd/project-server
.PHONY: build

build-image: build
	hack/build-image.sh
.PHONY: build-image

verify:
	go test github.com/openshift/kube-projects/pkg/...
.PHONY: verify

clean:
	rm -rf _output
.PHONY: clean

update-deps:
	hack/update-deps.sh
.PHONY: generate