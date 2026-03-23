BINARY     := github-authorized-keys
INSTALL_BIN := $(HOME)/.local/bin/$(BINARY)
SYSTEMD_DIR := $(HOME)/.config/systemd/user
SERVICE     := github-authorized-keys.service
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     := -ldflags "-X main.version=$(VERSION)"

.PHONY: build install uninstall test lint clean

## build: compile the binary for the current platform
build:
	go build $(LDFLAGS) -o $(BINARY) .

## install: build and install binary + systemd user service
install: build
	install -Dm755 $(BINARY) $(INSTALL_BIN)
	install -Dm644 systemd/$(SERVICE) $(SYSTEMD_DIR)/$(SERVICE)
	systemctl --user daemon-reload
	systemctl --user enable --now $(SERVICE)
	@echo "Installed and started $(SERVICE)"
	@echo "Check status with: systemctl --user status $(SERVICE)"

## uninstall: stop and remove the service and binary
uninstall:
	-systemctl --user disable --now $(SERVICE) 2>/dev/null
	rm -f $(INSTALL_BIN) $(SYSTEMD_DIR)/$(SERVICE)
	-systemctl --user daemon-reload 2>/dev/null
	@echo "Uninstalled $(BINARY)"

## test: run all tests
test:
	go test -v -race ./...

## lint: run go vet
lint:
	go vet ./...

## clean: remove build artifacts
clean:
	rm -f $(BINARY)

## help: display available targets
help:
	@grep -E '^##' Makefile | sed 's/## //'
