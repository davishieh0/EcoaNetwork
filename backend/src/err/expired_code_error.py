"""
Expired code error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class ExpiredCodeError(EcoaError):
    """
    Raised when a confirmation or reset code is no longer valid.
    """

    MESSAGE = 'Code expired!'
    STATUS_CODE = 400
    LOG_LEVEL = logging.WARNING
