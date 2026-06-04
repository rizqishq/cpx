WINDOWS_OUT_DIR := /mnt/d/tmp/cpx-test
LINUX_BIN := ./cpx
WINDOWS_BIN := $(WINDOWS_OUT_DIR)/cpx.exe

.PHONY: help build build-linux build-windows clean

help:
	@printf 'Targets:\n'
	@printf '  build          Build Linux/WSL and Windows binaries\n'
	@printf '  build-linux    Build Linux/WSL binary to $(LINUX_BIN)\n'
	@printf '  build-windows  Build Windows binary to $(WINDOWS_BIN)\n'
	@printf '  clean          Remove built artifacts from repo and $(WINDOWS_OUT_DIR)\n'

build: build-linux build-windows

build-linux:
	go build -o "$(LINUX_BIN)" .

build-windows:
	@mkdir -p "$(WINDOWS_OUT_DIR)"
	GOOS=windows GOARCH=amd64 go build -o "$(WINDOWS_BIN)" .

clean:
	rm -f "$(LINUX_BIN)" "$(WINDOWS_BIN)"
