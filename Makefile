.PHONY: build clean login gateway help

# 获取 Git 信息
VERSION=$(shell git describe --tags --always || echo "v0.0.0")
GIT_COMMIT=$(shell git rev-parse --short HEAD || echo "none")
BUILD_DATE=$(shell date "+%Y-%m-%d %H:%M:%S")

# 注入路径 (必须匹配 pkg/app/version.go 所在位置)
PKG_PATH=github.com/lk2023060901/xdooria/pkg/app

# LDFLAGS
LDFLAGS=-ldflags "-X '$(PKG_PATH).Version=$(VERSION)' \
                  -X '$(PKG_PATH).GitCommit=$(GIT_COMMIT)' \
                  -X '$(PKG_PATH).BuildDate=$(BUILD_DATE)' \
                  -X '$(PKG_PATH).AppName=login-svc'"

login:
	@echo "Building Login Service..."
	cd app/login/cmd && wire gen
	go build $(LDFLAGS) -o bin/login-svc ./app/login/cmd
	@cp app/login/cmd/config.yaml bin/config.yaml 2>/dev/null || true
	@mkdir -p bin/configs/data
	@cp app/login/cmd/configs/data/*.json bin/configs/data/ 2>/dev/null || true
	@echo "Done. Binary: bin/login-svc"

