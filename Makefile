.PHONY: all linux windows mac rpi clean version

NAME := "gitrip"
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.0.0")
BUILD_DATE := $(shell date -u +%y%m%d)
GIT_HASH := $(shell git rev-parse --short=4 HEAD)
BUILD_VERSION := $(VERSION)-$(GIT_HASH)-$(BUILD_DATE)
LDFLAGS := -X 'main.buildVersion=$(BUILD_VERSION)'

all: version clean linux windows mac

linux:
	@echo "Linux"
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/$(NAME).amd64.elf cmd/gitrip.go
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/$(NAME).arm64.elf cmd/gitrip.go

windows:
	@echo "Windows"
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/$(NAME).exe cmd/gitrip.go

mac:
	@echo "MacOS"
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/$(NAME).amd64.mac cmd/gitrip.go
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/$(NAME).arm64.mac cmd/gitrip.go

clean:
	@echo "Cleaning up"
	rm -f bin/$(NAME)*

version:
	@echo $(BUILD_VERSION)
