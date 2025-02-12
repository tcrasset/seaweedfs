#!/usr/bin/env python3
# /// script
# dependencies = [
#   "boto3>1.36",
# ]
# ///


import base64
import importlib
import importlib.metadata
import os
import tempfile

import boto3


def main():
    os.environ["AWS_ACCESS_KEY_ID"] = "power_user_key" # from docker/compose/s3.json
    os.environ["AWS_SECRET_ACCESS_KEY"] = "power_user_secret" # from docker/compose/s3.json
    os.environ["AWS_ENDPOINT_URL"] = "http://localhost:8333"

    boto3_version = importlib.metadata.version("boto3")
    print(f"boto3 version: {boto3_version}")

    to_write = b"Hello World"* 1000000
    checksum = 1
    checksum = base64.b64encode(checksum.to_bytes(4, byteorder='big')).decode('utf-8')

    with tempfile.NamedTemporaryFile() as f:
        f.write(to_write)
        f.flush()

        client = boto3.client("s3")
        client.upload_file(Bucket="foo", Key="bar", Filename=f.name)




if __name__ == '__main__':
    main()