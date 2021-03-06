# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gaae android ios gaae-cross swarm evm all test clean
.PHONY: gaae-linux gaae-linux-386 gaae-linux-amd64 gaae-linux-mips64 gaae-linux-mips64le
.PHONY: gaae-linux-arm gaae-linux-arm-5 gaae-linux-arm-6 gaae-linux-arm-7 gaae-linux-arm64
.PHONY: gaae-darwin gaae-darwin-386 gaae-darwin-amd64
.PHONY: gaae-windows gaae-windows-386 gaae-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

gaae:
	build/env.sh go run build/ci.go install ./cmd/gaae
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gaae\" to launch gaae."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/gaae.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Gaae.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

gaae-cross: gaae-linux gaae-darwin gaae-windows gaae-android gaae-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gaae-*

gaae-linux: gaae-linux-386 gaae-linux-amd64 gaae-linux-arm gaae-linux-mips64 gaae-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gaae-linux-*

gaae-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/gaae
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gaae-linux-* | grep 386

gaae-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gaae
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gaae-linux-* | grep amd64

gaae-linux-arm: gaae-linux-arm-5 gaae-linux-arm-6 gaae-linux-arm-7 gaae-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gaae-linux-* | grep arm

gaae-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/gaae
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gaae-linux-* | grep arm-5

gaae-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/gaae
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gaae-linux-* | grep arm-6

gaae-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/gaae
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gaae-linux-* | grep arm-7

gaae-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/gaae
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gaae-linux-* | grep arm64

gaae-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/gaae
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/gaae-linux-* | grep mips

gaae-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/gaae
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/gaae-linux-* | grep mipsle

gaae-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/gaae
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/gaae-linux-* | grep mips64

gaae-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/gaae
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/gaae-linux-* | grep mips64le

gaae-darwin: gaae-darwin-386 gaae-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gaae-darwin-*

gaae-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/gaae
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gaae-darwin-* | grep 386

gaae-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gaae
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gaae-darwin-* | grep amd64

gaae-windows: gaae-windows-386 gaae-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gaae-windows-*

gaae-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/gaae
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gaae-windows-* | grep 386

gaae-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gaae
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gaae-windows-* | grep amd64
