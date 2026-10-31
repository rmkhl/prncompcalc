BINARY = prncompcalc
BINDIR = bin

# Workspace-local toolchain - every go and golangci-lint invocation uses this
# installation regardless of what is on the system. CI installs it the same way,
# so these two lines are the only place the versions are pinned. After changing
# one, run 'make distclean' to pick it up: the install rules only look at
# whether the binary exists.
GO_VERSION = 1.26.10
GOLANGCI_LINT_VERSION = 2.12.1

TOOLS_DIR = $(CURDIR)/.go
GO = $(TOOLS_DIR)/bin/go
GOLANGCI_LINT = $(TOOLS_DIR)/bin/golangci-lint

# Keep the module and build caches inside the project too, and stop the go
# command downloading some other toolchain behind the pinned one's back.
export PATH := $(TOOLS_DIR)/bin:$(PATH)
export GOROOT := $(TOOLS_DIR)
export GOPATH := $(TOOLS_DIR)/path
export GOCACHE := $(TOOLS_DIR)/cache/go-build
export GOLANGCI_LINT_CACHE := $(TOOLS_DIR)/cache/golangci-lint
export GOTOOLCHAIN := local

$(GO):
	@echo "Installing Go $(GO_VERSION) locally in .go/..."
	@mkdir -p $(TOOLS_DIR)
	@ARCH=$$(uname -m); \
	OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	case "$$ARCH" in \
		x86_64) GO_ARCH="amd64" ;; \
		aarch64|arm64) GO_ARCH="arm64" ;; \
		*) echo "Error: Unsupported architecture: $$ARCH"; exit 1 ;; \
	esac; \
	curl -fsSL "https://go.dev/dl/go$(GO_VERSION).$$OS-$$GO_ARCH.tar.gz" -o $(TOOLS_DIR)/go.tar.gz; \
	tar -xzf $(TOOLS_DIR)/go.tar.gz -C $(TOOLS_DIR) --strip-components=1; \
	rm $(TOOLS_DIR)/go.tar.gz
	@echo "✓ Go $(GO_VERSION) installed to .go/"

$(GOLANGCI_LINT): | $(GO)
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) locally in .go/bin/..."
	@ARCH=$$(uname -m); \
	OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	case "$$ARCH" in \
		x86_64) LINT_ARCH="amd64" ;; \
		aarch64|arm64) LINT_ARCH="arm64" ;; \
		*) echo "Error: Unsupported architecture: $$ARCH"; exit 1 ;; \
	esac; \
	LINT_DIST="golangci-lint-$(GOLANGCI_LINT_VERSION)-$$OS-$$LINT_ARCH"; \
	curl -fsSL "https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_LINT_VERSION)/$$LINT_DIST.tar.gz" -o $(TOOLS_DIR)/golangci-lint.tar.gz; \
	tar -xzf $(TOOLS_DIR)/golangci-lint.tar.gz -C $(TOOLS_DIR)/bin --strip-components=1 "$$LINT_DIST/golangci-lint"; \
	rm $(TOOLS_DIR)/golangci-lint.tar.gz
	@echo "✓ golangci-lint $(GOLANGCI_LINT_VERSION) installed to .go/bin/"

.PHONY: prepare
prepare: $(GO) $(GOLANGCI_LINT)
	@echo "✓ Workspace prepared"

.PHONY: all
all: clean $(BINDIR)/$(BINARY)
	@echo "✓ Build completed"

$(BINDIR)/$(BINARY): $(shell find cmd internal -name '*.go') go.mod | $(GO) $(BINDIR)
	@$(GO) build -o $@ ./cmd/$(BINARY)

$(BINDIR):
	@mkdir -p $(BINDIR)

.PHONY: clean
clean:
	@rm -rf $(BINDIR)
	@echo "✓ Cleaned Go binaries"

# The module cache is read-only by design, so it has to be made writable first.
.PHONY: distclean
distclean: clean
	@chmod -R u+w .go 2>/dev/null || true
	@rm -rf .go
	@echo "✓ Removed local Go toolchain and caches"

.PHONY: test
test: | $(GO)
	@$(GO) test ./...

.PHONY: lint
lint: | $(GOLANGCI_LINT)
	@$(GOLANGCI_LINT) run ./...

# Everything CI runs, in one command, for use before pushing. Reports every
# check rather than stopping at the first failure, then exits non-zero if any
# failed.
.PHONY: ci-check
ci-check:
	@failed=""; \
	$(MAKE) --no-print-directory test || failed="$$failed tests"; \
	$(MAKE) --no-print-directory lint || failed="$$failed lint"; \
	echo ""; \
	if [ -n "$$failed" ]; then \
		echo "✗ CI check failed:$$failed"; \
		exit 1; \
	fi; \
	echo "✓ All CI checks passed"

.PHONY: fmt
fmt: | $(GOLANGCI_LINT)
	@$(GOLANGCI_LINT) run --fix ./...

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  help              Show this help message (default)."
	@echo "  prepare           Install Go and golangci-lint locally in .go/."
	@echo ""
	@echo "Build Targets:"
	@echo "  all               Clean and build bin/$(BINARY)."
	@echo "  clean             Remove bin/."
	@echo "  distclean         As 'clean', plus removes the local toolchain in .go/."
	@echo ""
	@echo "Checks:"
	@echo "  test              Run the tests."
	@echo "  lint              Run golangci-lint."
	@echo "  fmt               Reformat Go files."
	@echo "  ci-check          Run everything CI runs, before pushing."

.DEFAULT_GOAL := help
