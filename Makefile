SHELL = /bin/bash
GO := go
OBJ := gh2jira
SPECIFIC_UNIT_TEST := $(if $(TEST),-run $(TEST),)
extra_env := $(GOENV)

# define characters
null  :=
space := $(null) #
comma := ,

.PHONY: all
all: clean unit build

$(OBJ):
	$(extra_env) $(GO) build $(extra_flags) -o $@ .

.PHONY: build
build: clean $(OBJ)

.PHONY: unit
unit:
	$(GO) test -coverprofile=coverage.out $(SPECIFIC_UNIT_TEST) -count=1 ./...

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: lint
lint:
	find . -name '*.go' | xargs goimports -w

.PHONY: clean
clean:
	@rm -rf $(OBJ)

