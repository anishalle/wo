BINARY := wo
CMD := ./cmd/wo
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
MANDIR ?= $(PREFIX)/share/man/man1
COMPLETION_DIR ?= ./completions
MAN_BUILD_DIR ?= ./man

.PHONY: build test install completion man clean

build:
	go build -o ./bin/$(BINARY) $(CMD)

test:
	go test ./...

completion:
	mkdir -p $(COMPLETION_DIR)
	go run $(CMD) completion bash > $(COMPLETION_DIR)/wo.bash
	go run $(CMD) completion zsh > $(COMPLETION_DIR)/_wo
	go run $(CMD) completion fish > $(COMPLETION_DIR)/wo.fish

man:
	mkdir -p $(MAN_BUILD_DIR)
	go run $(CMD) man --dir $(MAN_BUILD_DIR)

install: build completion man
	mkdir -p $(BINDIR)
	install -m 0755 ./bin/$(BINARY) $(BINDIR)/$(BINARY)
	mkdir -p $(MANDIR)
	install -m 0644 $(MAN_BUILD_DIR)/wo.1 $(MANDIR)/wo.1

clean:
	rm -rf ./bin $(COMPLETION_DIR) $(MAN_BUILD_DIR)
