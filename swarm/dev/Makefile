.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/aaechain/go-aaechain/cmd/swarm
	go build -o bin/gaae     github.com/aaechain/go-aaechain/cmd/gaae
	go build -o bin/bootnode github.com/aaechain/go-aaechain/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/aaechain/go-aaechain/swarm/...
