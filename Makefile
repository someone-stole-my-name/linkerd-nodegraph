PROJECT = linkerd-nodegraph
GOCMD := go
GOTEST = $(GOCMD) test -v -race -cover
GOBUILD = $(GOCMD) build -buildvcs=false -mod vendor
GOVERSION = 1.19

IMAGE_NAME := ghcr.io/someone-stole-my-name/$(PROJECT)
IMAGE_PLATFORMS := linux/amd64,linux/arm64

define DOCKER_DEPS
	binfmt-support \
	ca-certificates \
	curl \
	git \
	gnupg \
	jq \
	lsb-release \
	make \
	qemu-user-static \
	wget
endef


all: clean vendor test build

clean:
	$(RM) -r out

vendor:
	$(GOCMD) mod tidy
	$(GOCMD) mod vendor 

test:
	$(GOTEST) ./...

build:
	mkdir -p out/bin
	CGO_ENABLED=0 $(GOBUILD) -o out/bin/nodegraph-server ./cmd/nodegraph-server

setup-buildx:
	curl -fsSL https://download.docker.com/linux/debian/gpg |\
		gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg && \
	echo "deb [arch=$(shell dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian $(shell lsb_release -cs) stable" |\
	tee /etc/apt/sources.list.d/docker.list > /dev/null && \
	cat /etc/apt/sources.list.d/docker.list && \
	apt-get update && \
	apt-get -qq -y install \
		docker-ce \
		docker-ce-cli \
		containerd.io
	docker run --privileged --rm tonistiigi/binfmt --install all
	docker buildx create --name mybuilder
	docker buildx use mybuilder

push: setup-buildx
	docker buildx build \
		--build-arg GOVERSION=$(GOVERSION) \
		--platform $(IMAGE_PLATFORMS) \
		-t $(IMAGE_NAME):latest . --push
	if git describe --exact-match; then	\
			docker buildx build \
				--build-arg GOVERSION=$(GOVERSION) \
				--platform $(IMAGE_PLATFORMS) \
				-t $(IMAGE_NAME):$(shell git describe --tags --abbrev=0) . --push; \
	fi

export DOCKER_DEPS
docker-%:
	docker run \
		--rm \
		--privileged \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(shell pwd):/data \
		-w /data $(DOCKER_EXTRA_ARGS) \
		golang:$(GOVERSION) sh -c "\
			apt-get update && \
			apt-get install -y $$DOCKER_DEPS && make $*"
