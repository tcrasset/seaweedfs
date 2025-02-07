#!/usr/bin/env python3
# /// script
# dependencies = [
#   "requests<3",
#   "structlog",
#   "pandas",
#   "fsspec",
#   "s3fs",
#   "pyyaml",
#   "typer<0.15.1",
# ]
# ///
import logging
import logging.config
import os
import sys

import structlog
import typer

LOG_LEVEL = str(os.getenv("LOG_LEVEL", "DEBUG")).upper()


def setup_logging(log_level=LOG_LEVEL):
    timestamper = structlog.processors.TimeStamper(fmt="%Y-%m-%d %H:%M:%S")
    structlog.configure(
        processors=[
            structlog.stdlib.add_log_level,
            structlog.stdlib.add_logger_name,
            structlog.stdlib.PositionalArgumentsFormatter(),
            timestamper,
            structlog.processors.StackInfoRenderer(),
            structlog.processors.format_exc_info,
            structlog.stdlib.ProcessorFormatter.wrap_for_formatter,
            # add logger name to the event_dict
        ],
        logger_factory=structlog.stdlib.LoggerFactory(),
        wrapper_class=structlog.stdlib.BoundLogger,
        cache_logger_on_first_use=True,
    )

    pre_chain = [
        # Add the name of the logger to event dict.
        structlog.stdlib.add_logger_name,
        # Add the log level and a timestamp to the event_dict if the log entry
        # is not from structlog.
        structlog.stdlib.add_log_level,
        # Add extra attributes of LogRecord objects to the event dictionary
        # so that values passed in the extra parameter of log methods pass
        # through to log output.
        structlog.stdlib.ExtraAdder(),
        timestamper,
    ]

    logging.config.dictConfig(
        {
            "version": 1,
            "disable_existing_loggers": False,
            "formatters": {
                "plain": {
                    "()": structlog.stdlib.ProcessorFormatter,
                    "processors": [
                        structlog.stdlib.ProcessorFormatter.remove_processors_meta,
                        structlog.dev.ConsoleRenderer(colors=False),
                    ],
                    "foreign_pre_chain": pre_chain,
                },
                "colored": {
                    "()": structlog.stdlib.ProcessorFormatter,
                    "processors": [
                        structlog.stdlib.ProcessorFormatter.remove_processors_meta,
                        structlog.dev.ConsoleRenderer(colors=True),
                    ],
                    "foreign_pre_chain": pre_chain,
                },
            },
            "handlers": {
                "default": {
                    "level": log_level,
                    "class": "logging.StreamHandler",
                    "formatter": "colored",
                }
            },
            "loggers": {
                "": {
                    "handlers": ["default"],
                    "level": log_level,
                    "propagate": True,
                },
            },
        }
    )



setup_logging()

logger = structlog.get_logger(__name__)



def get_filesystem():
    s3_fs_kwargs = {
        "config_kwargs":{
            'signature_version': 's3v4'
        }

    }
    import s3fs

    fsspec_fs = s3fs.S3FileSystem(anon=False, **s3_fs_kwargs)
    return fsspec_fs








def validate_aws_keys():
    if "AWS_ENDPOINT_URL" not in os.environ or "AWS_ACCESS_KEY_ID" not in os.environ or "AWS_SECRET_ACCESS_KEY" not in os.environ:
        logger.error("AWS credentials not set")
        sys.exit(1)

def upload(filepath: str):
    validate_aws_keys()
    fsspec_fs = get_filesystem(
    )
    bucket_name = "64b2cff7-d7f6-4421-b0e8-6887a10d5009"
    # with suppress(FileExistsError):

    #     fsspec_fs.mkdir(bucket_name)

    dataset_input_url = f"s3://{bucket_name}/file.txt"
    print(f"Uploading dataset to {dataset_input_url}")
    with open(filepath, "rb") as input_f:
        with fsspec_fs.open(dataset_input_url, "wb") as output_f:
            output_f.write(input_f.read())


if __name__ == "__main__":
    typer.run(upload)