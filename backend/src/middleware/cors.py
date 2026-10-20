"""
CORS configuration.
"""

# --- IMPORTS ---
from collections.abc import Iterable
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from src.config import config

import re


# --- CODE ---
def origin_regex(origins: Iterable[str]) -> str | None:
    """
    Builds the regex matching every ':*' origin on any port.

    :param origins: The configured origins.

    :return: The regex, or None when no origin uses ':*'.
    """
    hosts = [re.escape(o[:-2]) for o in origins if o.endswith(':*')]

    # no wildcard origin: nothing to match
    if not hosts:
        return None

    return f'^({"|".join(hosts)}):\\d+$'


def add_cors(app: FastAPI) -> None:
    """
    Adds the CORS middleware to the application.

    :param app: The FastAPI application instance.

    :return: None.
    """
    origins = config.cors_origins

    app.add_middleware(
        CORSMiddleware,
        allow_origins=[o for o in origins if not o.endswith(':*')],
        allow_origin_regex=origin_regex(origins),
        allow_credentials=True,
        allow_methods=['*'],
        allow_headers=['*'],
    )
