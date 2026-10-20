"""
Profile exists error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class ProfileExistsError(EcoaError):
    """
    Raised when an account that already has a profile asks for another. The
    handle is chosen once.
    """

    MESSAGE = 'Profile already exists!'
    STATUS_CODE = 409
    LOG_LEVEL = logging.WARNING
