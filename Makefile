SHELL := /bin/bash
CURR_DIR = $(shell pwd)

linter-image:
	docker build --rm -f Dockerfile.mdlinter -t danielguo/mdlinter .

md-linter:
	docker run -v ${CURR_DIR}:/workdir danielguo/mdlinter