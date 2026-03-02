BINARY := wo
CMD := ./cmd/wo
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
MANDIR ?= $(PREFIX)/share/man/man1
COMPLETION_DIR ?= ./completions
MAN_BUILD_DIR ?= ./man

.PHONY: build test install uninstall completion man clean

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

uninstall:
	rm -f $(BINDIR)/$(BINARY)
	rm -f $(MANDIR)/wo.1
	@set -eu; \
	remove_db() { \
		dir="$$1"; \
		[ -n "$$dir" ] || return 0; \
		rm -f "$$dir/wo.db" "$$dir/wo.db-shm" "$$dir/wo.db-wal" "$$dir/wo.db-journal"; \
		rmdir "$$dir" 2>/dev/null || true; \
	}; \
	remove_db "$$HOME/Library/Caches/wo"; \
	if [ -n "$${XDG_DATA_HOME:-}" ]; then remove_db "$${XDG_DATA_HOME}/wo"; fi; \
	remove_db "$$HOME/.cache/wo"; \
	remove_db "$$HOME/.config/wo"; \
	remove_db "$$HOME/Library/Application Support/wo"
	@echo "Uninstalled wo binary/man page and removed runtime index DB state."
	@echo "Kept config files and left workspace .wo files untouched."

clean:
	rm -rf ./bin $(COMPLETION_DIR) $(MAN_BUILD_DIR)
