#!/usr/bin/env bash

export AWS_ACCESS_KEY_ID=accesskey
export AWS_SECRET_ACCESS_KEY=secretkey
export AWS_DEFAULT_REGION=us-west-2
export AWS_REGION=us-west-2

PORT=18080
HOST=localhost

aws kms --endpoint-url http://$HOST:$PORT "$@"