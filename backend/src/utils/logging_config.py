"""
Logging configuration for the application.
"""

# --- IMPORTS ---
from src.config import config

import colorlog
import logging


# --- CODE ---
def setup_logger() -> logging.Logger:
    """
    Set up the root logger.

    :return: Configured root logger.
    """
    # get log level from config, default to DEBUG if not a known level
    log_level = logging.getLevelNamesMapping().get(
        config.LOG_LEVEL.upper(), logging.DEBUG
    )

    # create color formatter
    formatter = colorlog.ColoredFormatter(
        '%(white)s%(asctime)s%(reset)s '
        '%(log_color)s%(levelname)-8s%(reset)s '
        '%(cyan)s[%(name)s]%(reset)s %(message)s',
        datefmt='%Y-%m-%d %H:%M:%S',
        log_colors={
            'DEBUG': 'cyan',
            'INFO': 'green',
            'WARNING': 'yellow',
            'ERROR': 'red',
            'CRITICAL': 'red,bg_white',
        },
    )

    # create the stream handler
    handler = logging.StreamHandler()
    handler.setFormatter(formatter)

    # configure root logger, replacing any existing handlers
    root_logger = logging.getLogger()
    root_logger.setLevel(log_level)
    root_logger.handlers = [handler]

    # silence noisy external libraries
    logging.getLogger('httpx').setLevel(logging.WARNING)
    logging.getLogger('httpcore').setLevel(logging.WARNING)
    logging.getLogger('asyncio').setLevel(logging.WARNING)
    logging.getLogger('botocore').setLevel(logging.WARNING)
    logging.getLogger('aiobotocore').setLevel(logging.WARNING)

    # return the configured root logger
    return root_logger
