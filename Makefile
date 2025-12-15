.PHONY: build install clean

BINARY_NAME=azssh
INSTALL_PATH=/usr/local/bin

build:
	go build -o $(BINARY_NAME)

install: build
	sudo install -m 755 $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "$(BINARY_NAME) installed to $(INSTALL_PATH)/$(BINARY_NAME)"

uninstall:
	sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "$(BINARY_NAME) removed from $(INSTALL_PATH)"

clean:
	rm -f $(BINARY_NAME)
	@echo "Build artifacts cleaned"

all: build
