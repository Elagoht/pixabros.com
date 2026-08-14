DATA_DIR ?= ./data
PANEL_DIR = internal/adminui/dist

.PHONY: admin-build admin-embed build release-linux run dev test

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

# What gets shipped to a Debian server: the binary cross-compiled for it, with
# the installer and the unit beside it in the layout install.sh expects.
#
# CGO is off, so the result is static and depends on no glibc version at all --
# nothing here needs it, because the SQLite driver is Go rather than a C
# library wrapped in it. -trimpath keeps this machine's paths out of the
# binary; -s -w drops the debug tables, which Go stack traces do not use.
RELEASE_DIR = dist/pixabros-linux-amd64

release-linux: admin-embed
	rm -rf $(RELEASE_DIR)
	mkdir -p $(RELEASE_DIR)/deploy
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -trimpath -ldflags "-s -w" -o $(RELEASE_DIR)/pixabros ./cmd/server
	cp deploy/install.sh deploy/pixabros.service $(RELEASE_DIR)/deploy/
	chmod +x $(RELEASE_DIR)/deploy/install.sh
	@echo
	@echo "Built $(RELEASE_DIR):"
	@ls -la $(RELEASE_DIR) $(RELEASE_DIR)/deploy

run:
	go run ./cmd/server

# dev skips the npm install and reuses whatever panel build is already
# embedded, which is what you want when only Go code changed.
dev:
	go build -o pixabros ./cmd/server && ./pixabros

test:
	go test ./...
	npm test
	npm --prefix admin-ui run test
