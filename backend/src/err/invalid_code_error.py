"""
Invalid code error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class InvalidCodeError(EcoaError):
    """
    Raised when a confirmation or reset code does not match.
    """

    MESSAGE = 'Invalid code!'
    STATUS_CODE = 400
    LOG_LEVEL = logging.WARNING
