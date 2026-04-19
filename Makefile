BINARY  := umpire
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

.PHONY: build test run clean release

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test ./...

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)

release:
	@latest=$$(git tag --sort=-v:refname | head -1); \
	if [ -z "$$latest" ]; then \
		echo "No existing tags found. Creating v0.1.0"; \
		next="v0.1.0"; \
	else \
		major=$$(echo "$$latest" | sed 's/v//' | cut -d. -f1); \
		minor=$$(echo "$$latest" | sed 's/v//' | cut -d. -f2); \
		if [ "$(MAJOR)" = "1" ]; then \
			next="v$$((major + 1)).0.0"; \
		else \
			next="v$$major.$$((minor + 1)).0"; \
		fi; \
	fi; \
	echo "$$latest -> $$next"; \
	git tag "$$next" && \
	git push origin "$$next" && \
	echo "Released $$next"
