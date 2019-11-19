#!/bin/bash
# product-data-service

echo -e "  \e[2mGo \e[0m\e[94mBuild(ing)...\e[0m"
CGO_ENABLED=1 GO111MODULE=on go build -ldflags '-w' -a -o ./product-data-service

