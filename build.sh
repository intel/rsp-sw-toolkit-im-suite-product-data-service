#!/bin/bash
# mapping sku service

echo -e "  \e[2mGo \e[0m\e[94mBuild(ing)...\e[0m"
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo

