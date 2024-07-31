#!/bin/bash
set -e
YYYYMM=`date '+%Y%m'`
YYYYMMDD=`date '+%Y%m%d'`

# BUCKET_URL # from env (ex: "https://s3.ap-northeast-1.wasabisys.com")
# BUCKET_NAME # from env (ex: hoge-system-stg-bucket)
# BUCKET_DIR # from env (ex: fetcher/moneyforward)
# AWS_REGION # from env (ex: ap-northeast-1)
# AWS_ACCESS_KEY_ID # from env
# AWS_SECRET_ACCESS_KEY # from env
# user="xxxxxxxxx" # moneyforward id  , from env
# pass="yyyyyyyyy" # moneyforward pass, from env
# wsAddr # from env (ex: localhost:7327)

SCRAPERS_BIN="/usr/local/bin/myscrapers"
AWS_BIN="/usr/local/bin/aws/dist/aws"
outputDir="/data/${YYYYMM}/${YYYYMMDD}"

REMOTE_DIR="${BUCKET_DIR}/${YYYYMM}/${YYYYMMDD}"

function download () {
    echo "job start"
    mkdir -p ${outputDir}
    echo "output to dir: ${outputDir}"
    outputDir=${outputDir} \
    user=${user} \
    pass=${pass} \
    ${SCRAPERS_BIN} download moneyforward --lastmonth
    echo "job complete"
}

function create_s3_credentials () {
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
    ${AWS_BIN} s3 cp ${outputDir}/ "s3://${BUCKET_NAME}/${REMOTE_DIR}" --recursive --endpoint-url="${BUCKET_URL}"
    echo "s3 upload complete"
}

download

if [ -n $BUCKET_NAME ]; then
    create_s3_credentials
    s3_upload
fi
