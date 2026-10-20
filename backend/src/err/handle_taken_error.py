"""
Handle taken error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class HandleTakenError(EcoaError):
    """
    Raised when another account already uses the handle.
    """

    MESSAGE = 'Handle already taken!'
    STATUS_CODE = 409
    LOG_LEVEL = logging.WARNING
