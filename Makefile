SHELL := /bin/bash
CURR_DIR = $(shell pwd)

md-linter:
	docker run -v ${CURR_DIR}:/workdir danielguo/mdlinter