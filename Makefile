DATA_DIR ?= ./data

.PHONY: admin-build admin-deploy run test

admin-build:
	npm --prefix admin-ui ci
	npm --prefix admin-ui run build

# The server serves the SPA from $(DATA_DIR)/admin-dist, not admin-ui/dist,
# so a build is only live once it has been copied across.
admin-deploy: admin-build
	rm -rf $(DATA_DIR)/admin-dist
	mkdir -p $(DATA_DIR)/admin-dist
	cp -R admin-ui/dist/. $(DATA_DIR)/admin-dist/

run:
	go run ./cmd/server

test:
	go test ./...
