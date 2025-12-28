.PHONY: build clean login game gateway help

# 获取 Git 信息
VERSION=$(shell git describe --tags --always || echo "v0.0.0")
GIT_COMMIT=$(shell git rev-parse --short HEAD || echo "none")
BUILD_DATE=$(shell date "+%Y-%m-%d %H:%M:%S")

# 注入路径 (必须匹配 pkg/app/version.go 所在位置)
PKG_PATH=github.com/lk2023060901/xdooria/pkg/app

# LDFLAGS for login
LDFLAGS_LOGIN=-ldflags "-X '$(PKG_PATH).Version=$(VERSION)' \
                  -X '$(PKG_PATH).GitCommit=$(GIT_COMMIT)' \
                  -X '$(PKG_PATH).BuildDate=$(BUILD_DATE)' \
                  -X '$(PKG_PATH).AppName=login-svc'"

# LDFLAGS for game
LDFLAGS_GAME=-ldflags "-X '$(PKG_PATH).Version=$(VERSION)' \
                  -X '$(PKG_PATH).GitCommit=$(GIT_COMMIT)' \
                  -X '$(PKG_PATH).BuildDate=$(BUILD_DATE)' \
                  -X '$(PKG_PATH).AppName=game-svc'"

login:
	@echo "Building Login Service..."
	cd app/login/cmd && wire gen
	go build $(LDFLAGS_LOGIN) -o bin/login-svc ./app/login/cmd
	@cp app/login/cmd/config.yaml bin/config.yaml 2>/dev/null || true
	@mkdir -p bin/configs/data
	@cp app/login/cmd/configs/data/*.json bin/configs/data/ 2>/dev/null || true
	@echo "Done. Binary: bin/login-svc"

game:
	@echo "Building Game Service..."
	cd app/game/cmd && wire gen
	go build $(LDFLAGS_GAME) -o bin/game-svc ./app/game/cmd
	@cp app/game/cmd/config.yaml bin/game-config.yaml 2>/dev/null || true
	@mkdir -p bin/game-configs/data
	@cp app/game/cmd/configs/data/*.json bin/game-configs/data/ 2>/dev/null || true
	@echo "Done. Binary: bin/game-svc"

