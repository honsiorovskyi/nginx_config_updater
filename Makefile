BUILD_DIR = .build
REPO_NAME = github.com/honsiorovskyi/nginx_config_updater
BIN_PATH = $(BUILD_DIR)/$(REPO_NAME)
SOURCES = \
		  config.go \
		  err.go \
		  http.go \
		  updater.go

$(BIN_PATH): $(SOURCES)
	go build -o $(BIN_PATH) .
