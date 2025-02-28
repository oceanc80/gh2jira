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
all: unit build

$(OBJ):
	$(extra_env) $(GO) build $(extra_flags) -o $@ .

.PHONY: build
build: $(OBJ)

.PHONY: unit
unit:
	$(GO) test -coverprofile=coverage.out $(SPECIFIC_UNIT_TEST) -count=1 ./...

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: lint
lint:
	find . -type f -name '*.go' | xargs gosimports -w -local $(shell go list -m)

.PHONY: clean
clean:
	@rm -rf $(OBJ) coverage.out

.PHONY: sanity
sanity: fmt vet lint
	git diff --exit-code

