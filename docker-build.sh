#!/bin/bash
BUILD=`cat build`
ANTIGA=$((BUILD-1))
#docker rmi 505710261882.dkr.ecr.sa-east-1.amazonaws.com/reverse-proxy:build-$ANTIGA
#docker build -t 505710261882.dkr.ecr.sa-east-1.amazonaws.com/reverse-proxy:build-$BUILD .
docker build -t 505710261882.dkr.ecr.sa-east-1.amazonaws.com/reverse-proxy:devel .
docker push 505710261882.dkr.ecr.sa-east-1.amazonaws.com/reverse-proxy:devel
