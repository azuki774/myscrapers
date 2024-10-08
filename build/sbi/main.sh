#!/bin/bash
set -e
YYYYMM=`date '+%Y%m'`
YYYYMMDD=`date '+%Y%m%d'`

# BUCKET_URL # from env (ex: "https://s3.ap-northeast-1.wasabisys.com")
# BUCKET_NAME # from env (ex: hoge-system-stg-bucket)
# BUCKET_DIR # from env (ex: fetcher/sbi)
# AWS_REGION # from env (ex: ap-northeast-1)
# AWS_ACCESS_KEY_ID # from env
# AWS_SECRET_ACCESS_KEY # from env

SCRAPERS_BIN="/usr/local/bin/myscrapers"
AWS_BIN="/usr/local/bin/aws/dist/aws"
DATA_DIR="/data"

REMOTE_DIR="${BUCKET_DIR}/${YYYYMM}"

function fetch () {
    echo "fetcher start"
    python3 -u /src/main.py
    echo "fetcher complete"
}

function create_s3_credentials {
    echo "s3 credentials create start"
    mkdir -p ~/.aws/

    echo "[default]" >> ~/.aws/config
    echo "region = ${AWS_REGION}" >> ~/.aws/config

    echo "[default]" >> ~/.aws/credentials
    echo "aws_access_key_id = ${AWS_ACCESS_KEY_ID}" >> ~/.aws/credentials
    echo "aws_secret_access_key = ${AWS_SECRET_ACCESS_KEY}" >> ~/.aws/credentials

    chmod 400 ~/.aws/config
    chmod 400 ~/.aws/credentials
    ls -la ~/.aws/
    echo "s3 credentials create complete"
}

function s3_upload () {
    echo "s3 upload start"
    mkdir -p ${DATA_DIR}/${YYYYMM}
    cp -f ${DATA_DIR}/*.csv ${DATA_DIR}/${YYYYMM}/ # ex. $DATA_DIR/YYYYMMDD_1.csv -> $DATA_DIR/$YYYYMM/YYYYMMDD_1.csv
    rm ${DATA_DIR}/*.csv
    ${AWS_BIN} s3 cp ${DATA_DIR}/${YYYYMM}/ "s3://${BUCKET_NAME}/${REMOTE_DIR}" --recursive --endpoint-url="${BUCKET_URL}"
    echo "s3 upload complete"
}

fetch

if [ -z $BUCKET_NAME ]; then
    exit 0
fi

create_s3_credentials
s3_upload
