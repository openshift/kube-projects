all: build
.PHONY: all

build:
	go build -o _output/bin/kube-projects github.com/openshift/kube-projects/cmd/project-server
.PHONY: build

clean:
	rm -rf _output
.PHONY: clean

update-deps:
	hack/update-deps.sh
.PHONY: generate