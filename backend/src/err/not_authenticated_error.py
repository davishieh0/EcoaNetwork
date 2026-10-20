"""
Not authenticated error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class NotAuthenticatedError(EcoaError):
    """
    Raised when a request has no valid access token.
    """

    MESSAGE = 'Not authenticated!'
    STATUS_CODE = 401
    LOG_LEVEL = logging.WARNING
