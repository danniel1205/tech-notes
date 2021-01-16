
IMG_NAME ?= danielguo/test-app-template
IMG_TAG ?= latest

.PHONY: templates
templates:
	docker build -t $(IMG_NAME):$(IMG_TAG) -f Dockerfile.templates .
	docker push $(IMG_NAME):$(IMG_TAG)