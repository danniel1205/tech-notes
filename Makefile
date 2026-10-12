SHELL := /bin/bash
CURR_DIR = $(shell pwd)

linter-image:
	docker build --rm -f Dockerfile.mdlinter -t danielguo/mdlinter .

md-linter:
	docker run --rm -v ${CURR_DIR}:/workdir danielguo/mdlinter

md-fix:
	docker run --rm -v ${CURR_DIR}:/workdir danielguo/mdlinter --fix "**/*.md" "#node_modules"

toc:
	python3 generate_toc.py