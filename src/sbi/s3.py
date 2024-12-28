import boto3
from pathlib import Path
import os

def upload_files(dir_path):
    client = boto3.client(
        's3',
        endpoint_url=os.getenv("BUCKET_URL"),
        aws_access_key_id = os.getenv("AWS_ACCESS_KEY_ID"),
        aws_secret_access_key = os.getenv("AWS_SECRET_ACCESS_KEY"),
        region_name = os.getenv("AWS_REGION")
    )

    # dir_path ディレクトリ内のファイルを列挙
    os.chdir(dir_path) 
    for root, dirs, files in os.walk(dir_path):
        for f in files: # f:
            fullpath = os.path.join(root, f)
            relpath = Path(fullpath).relative_to(Path.cwd()) # s3アップロード時に dir_path そのもののパスは消すために移動
            print(relpath)
            client.upload_file(relpath, os.getenv("BUCKET_NAME"), os.path.join(os.getenv("BUCKET_DIR"), relpath))
