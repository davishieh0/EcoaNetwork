"""
Too many requests error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class TooManyRequestsError(EcoaError):
    """
    Raised when the identity provider is throttling the caller.
    """

    MESSAGE = 'Too many attempts, try again later!'
    STATUS_CODE = 429
    LOG_LEVEL = logging.WARNING
