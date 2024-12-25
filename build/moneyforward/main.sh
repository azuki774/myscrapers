#!/bin/bash
set -e

# BUCKET_URL # from env (ex: "https://s3.ap-northeast-1.wasabisys.com")
# BUCKET_NAME # from env (ex: hoge-system-stg-bucket)
# BUCKET_DIR # from env (ex: fetcher/moneyforward)
# AWS_REGION # from env (ex: ap-northeast-1)
# AWS_ACCESS_KEY_ID # from env
# AWS_SECRET_ACCESS_KEY # from env
# user="xxxxxxxxx" # moneyforward id  , from env
# pass="yyyyyyyyy" # moneyforward pass, from env

AWS_BIN="/usr/local/bin/aws/dist/aws"
DATA_DIR="/data"
REMOTE_DIR="${BUCKET_DIR}"

function fetch () {
    echo "fetcher start"
    python3 -u /src/main.py --s3-upload
    echo "fetcher complete"
}

fetch
