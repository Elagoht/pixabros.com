.PHONY: admin-build
admin-build:
	npm --prefix admin-ui ci
	npm --prefix admin-ui run build
