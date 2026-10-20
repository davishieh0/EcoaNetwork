"""
Profile incomplete error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class ProfileIncompleteError(EcoaError):
    """
    Raised when a signed-in account without a handle reaches a route that
    needs one: the client should ask for a handle first.
    """

    MESSAGE = 'Profile incomplete!'
    STATUS_CODE = 403
    LOG_LEVEL = logging.INFO
