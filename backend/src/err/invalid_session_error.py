"""
Invalid session error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class InvalidSessionError(EcoaError):
    """
    Raised when a session can no longer be refreshed.
    """

    MESSAGE = 'Session expired!'
    STATUS_CODE = 401
    LOG_LEVEL = logging.WARNING
