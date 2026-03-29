CURDIR    := $(abspath .)
OUT_PATH  := $(CURDIR)/pkg/proto_pb
LOCAL_BIN := $(CURDIR)/bin

ifeq ($(OS),Windows_NT)
  SHELL := cmd
  .SHELLFLAGS := /C
  EXE := .exe
  WIN_LOCAL_BIN := $(subst /,\\,$(LOCAL_BIN))
  MKDIR_P := if not exist "$(WIN_LOCAL_BIN)" mkdir "$(WIN_LOCAL_BIN)"
  BUF := $(WIN_LOCAL_BIN)\\buf$(EXE)
else
  EXE :=
  MKDIR_P := mkdir -p
  BUF := $(LOCAL_BIN)/buf
endif

.PHONY: bin-deps generate update linter linter-fix goose-up goose-down

# Установка buf + локальных плагинов
bin-deps:
ifeq ($(OS),Windows_NT)
	@$(MKDIR_P)
	@IF NOT EXIST "$(BUF)" ( \
	  ECHO == Installing buf into $(WIN_LOCAL_BIN) & \
	  set "GOBIN=$(WIN_LOCAL_BIN)" && go install github.com/bufbuild/buf/cmd/buf@latest || (ECHO !! buf install failed & exit /b 1) \
	) ELSE ( ECHO == buf already installed at $(BUF) )
	@ECHO == Installing local protoc plugins into $(WIN_LOCAL_BIN)
	@set "GOBIN=$(WIN_LOCAL_BIN)" && go install google.golang.org/protobuf/cmd/protoc-gen-go@latest || (ECHO !! protoc-gen-go failed & exit /b 1)
	@set "GOBIN=$(WIN_LOCAL_BIN)" && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest || (ECHO !! protoc-gen-go-grpc failed & exit /b 1)
	@REM ВАЖНО: PGV ставим из envoyproxy, это корректный module path
	@set "GOBIN=$(WIN_LOCAL_BIN)" && go install github.com/envoyproxy/protoc-gen-validate@latest || (ECHO !! protoc-gen-validate failed & exit /b 1)
	@set "GOBIN=$(WIN_LOCAL_BIN)" && go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest || (ECHO !! grpc-gateway failed & exit /b 1)
	@set "GOBIN=$(WIN_LOCAL_BIN)" && go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest || (ECHO !! openapiv2 failed & exit /b 1)
else
	@$(MKDIR_P) $(LOCAL_BIN)
	@if [ ! -x "$(BUF)" ]; then \
	  echo "== Installing buf into $(LOCAL_BIN)"; \
	  GOBIN=$(LOCAL_BIN) go install github.com/bufbuild/buf/cmd/buf@latest || exit 1; \
	else \
	  echo "== buf already installed at $(BUF)"; \
	fi
	@echo "== Installing local protoc plugins into $(LOCAL_BIN)"
	@GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@latest || exit 1
	@GOBIN=$(LOCAL_BIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest || exit 1
	@# ВАЖНО: PGV из envoyproxy
	@GOBIN=$(LOCAL_BIN) go install github.com/envoyproxy/protoc-gen-validate@latest || exit 1
	@GOBIN=$(LOCAL_BIN) go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest || exit 1
	@GOBIN=$(LOCAL_BIN) go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest || exit 1
endif

.PHONY: buf-deps

buf-deps:
ifeq ($(OS),Windows_NT)
	@echo == Updating buf dependencies ==
	@cd $(CURDIR) && set "PATH=$(WIN_LOCAL_BIN);%PATH%" && "$(BUF)" dep update
else
	@echo "== Updating buf dependencies =="
	cd $(CURDIR) && PATH="$(LOCAL_BIN):$$PATH" "$(BUF)" dep update
endif


generate-api: bin-deps
ifeq ($(OS),Windows_NT)
	@echo == Windows: generating pb ==
	@cd $(CURDIR) && set "PATH=$(WIN_LOCAL_BIN);%PATH%" && "$(BUF)" generate --template buf.gen.yaml
	@echo == generation pb complete ==
else
	@echo == Unix: generating pb ==
	cd $(CURDIR) && PATH="$(LOCAL_BIN):$$PATH" "$(BUF)" generate --template buf.gen.yaml
	@echo == generation pb complete ==
endif

generate: generate-api

# Обновить зависимости
update:
	go mod tidy

# Запустить линтер
linter:
	golangci-lint run

# Автофикс линтера
linter-fix:
	golangci-lint run --fix
