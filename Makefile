BUILD_DIR = .build
REPO_NAME = github.com/honsiorovskyi/nginx_config_updater
BIN_PATH = $(BUILD_DIR)/$(REPO_NAME)
SOURCES = \
		  config.go \
		  err.go \
		  http.go \
		  updater.go

VERSION=$(or $(TRAVIS_TAG), $(shell git describe --tags $(shell git rev-list --tags --max-count=1)))
BUILD_NUMBER=$(or $(TRAVIS_BUILD_NUMBER), 1)
COMMIT=$(or $(TRAVIS_COMMIT), $(shell git rev-parse --short HEAD))
GO_VERSION:=$(or $(TRAVIS_GO_VERSION), $(shell go version | awk '{ print $$3"-"$$4 }'))

.PHONY : clean

$(BIN_PATH) : $(SOURCES)
	go build -ldflags "-X main.Version=$(VERSION).$(BUILD_NUMBER)-$(COMMIT)-$(GO_VERSION)" -o $(BIN_PATH) .

clean :
	rm -rf $(BUILD_DIR)
