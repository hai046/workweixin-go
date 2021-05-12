#!/usr/bin/env bash

rm -rf bin
BASH_DIR=`pwd`

#build compile
cd cmd/app
GOARCH=amd64  GOOS=linux  go build -o  ../../bin/work-wechat-linux-amd64

#copy config file
cd ${BASH_DIR}
cp config.yaml bin/

#deploy
rsync -avzP bin/  root@test_host:/opt/projects/wechat-webhook

