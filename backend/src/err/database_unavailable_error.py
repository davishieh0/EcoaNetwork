"""
Database unavailable error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class DatabaseUnavailableError(EcoaError):
    """
    Raised when the database cannot be reached.
    """

    MESSAGE = 'Service temporarily unavailable!'
    STATUS_CODE = 503
    LOG_LEVEL = logging.ERROR
