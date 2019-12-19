# Apache v2 license
#  Copyright (C) <2019> Intel Corporation
#
#  SPDX-License-Identifier: Apache-2.0
#

.PHONY: build deploy stop init

MICROSERVICES=product-data-service 

BUILDABLE=$(MICROSERVICES)
.PHONY: $(BUILDABLE)

build: $(BUILDABLE)

$(MICROSERVICES):
	docker build --rm \
		--build-arg http_proxy=$(http_proxy) \
		--build-arg https_proxy=$(https_proxy) \
		-f Dockerfile_dev \
		-t rsp/$@:dev \
		.

deploy: init
	docker stack deploy \
		--with-registry-auth \
		--compose-file docker-compose.yml \
		Product-Data-Dev

init: 
	docker swarm init 2>/dev/null || true

stop:	
	docker stack rm Product-Data-Dev

