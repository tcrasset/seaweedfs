#!/usr/bin/env python3
# /// script
# dependencies = [
#   "botocore>1.36",
# ]


import os

import botocore
import botocore.session
from botocore.config import Config


def main():
    os.environ["AWS_ACCESS_KEY_ID"] = "any"
    os.environ["AWS_SECRET_ACCESS_KEY"] = "any"
    os.environ["AWS_ENDPOINT_URL"] = "http://localhost:8333"


    session = botocore.session.Session()

    client = session.create_client(
        "s3", "us-east-1", config=Config(signature_version="s3v4")
    )
    client.put_object(Bucket="foo", Key="bar", Body="a"*1024)




if __name__ == '__main__':
    main()