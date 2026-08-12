DATA_DIR ?= ./data
PANEL_DIR = internal/adminui/dist

.PHONY: admin-build admin-embed build run dev test

admin-build:
	npm --prefix admin-ui ci
	npm --prefix admin-ui run build

# The panel is embedded into the binary at compile time, so it has to be sitting
# in the Go package before `go build` runs. The placeholder index.html is
# replaced here and restored by git if the build output is ever removed.
admin-embed: admin-build
	rm -rf $(PANEL_DIR)
	mkdir -p $(PANEL_DIR)
	cp -R admin-ui/dist/. $(PANEL_DIR)/

# The whole deployment is one file: the binary carries the admin panel, the
# public site's templates, its stylesheet and its fonts.
build: admin-embed
	go build -o pixabros ./cmd/server

run:
	go run ./cmd/server

# dev skips the npm install and reuses whatever panel build is already
# embedded, which is what you want when only Go code changed.
dev:
	go build -o pixabros ./cmd/server && ./pixabros

test:
	go test ./...
	npm --prefix admin-ui run test
