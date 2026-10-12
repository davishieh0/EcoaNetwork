"""
Email already registered error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class EmailAlreadyRegisteredError(EcoaError):
    """
    Raised when an account already uses the email being registered.
    """

    MESSAGE = 'Email already registered!'
    STATUS_CODE = 409
    LOG_LEVEL = logging.WARNING
