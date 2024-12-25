import boto3
import os

def upload_file(filepath):
    client = boto3.client(
        's3',
        endpoint_url=os.getenv("BUCKET_URL"),
        aws_access_key_id = os.getenv("AWS_ACCESS_KEY_ID"),
        aws_secret_access_key = os.getenv("AWS_SECRET_ACCESS_KEY"),
        region_name = os.getenv("AWS_REGION")
    )
    basename = os.path.basename(filepath)
    client.upload_file(filepath, os.getenv("BUCKET_NAME"), os.getenv("BUCKET_DIR") + "/" + basename)
