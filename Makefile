BINARY_NAME=noctl
VERSION_FILE=VERSION
VERSION=v$(shell cat $(VERSION_FILE) | tr -d '[:space:]')
LDFLAGS=-X noctl/internal/version.Version=$(VERSION) -s -w
DIST_DIR=dist

.PHONY: all build clean build-all

all: build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

clean:
	rm -rf $(BINARY_NAME) $(BINARY_NAME).exe $(DIST_DIR)

build-all: clean
	mkdir -p $(DIST_DIR)
	# Linux amd64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME) .
	tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)
	rm $(DIST_DIR)/$(BINARY_NAME)
	# Linux arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME) .
	tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)
	rm $(DIST_DIR)/$(BINARY_NAME)
	# Darwin amd64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME) .
	tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)
	rm $(DIST_DIR)/$(BINARY_NAME)
	# Darwin arm64
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME) .
	tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)
	rm $(DIST_DIR)/$(BINARY_NAME)
	# Windows amd64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME).exe .
	zip -j $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(DIST_DIR)/$(BINARY_NAME).exe
	rm $(DIST_DIR)/$(BINARY_NAME).exe
