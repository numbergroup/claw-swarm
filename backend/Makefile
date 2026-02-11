# Pragma Services Makefile


BIN_DIR = bin
CMD_PATHS = $(wildcard cmd/*)
BINARIES=$(foreach x, $(notdir  $(CMD_PATHS)), ${BIN_DIR}/$(x))



.PHONY: $(BINARIES) all
all: bin-dir $(BINARIES)   ## Build all binaries


.PHONY: bin-dir
bin-dir:
	@mkdir -p $(BIN_DIR)

$(BINARIES): | bin-dir
	go build -o $@ ./cmd/$(notdir $@)
