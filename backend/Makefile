.PHONY: all clean test

include Makefile.build
include Makefile.defs
include Makefile.lint
include Makefile.format
include Makefile.run
include Makefile.testing

.PHONY: info
info:
	@echo "Go Arch: ${GOARCH}"
	@echo "Go OS: ${GOOS}"
	@echo "Service: ${PROJECT_NAME}"