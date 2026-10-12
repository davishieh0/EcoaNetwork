"""
Invalid password error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class InvalidPasswordError(EcoaError):
    """
    Raised when a new password does not meet the password policy.
    """

    MESSAGE = 'Password does not meet the requirements!'
    STATUS_CODE = 400
    LOG_LEVEL = logging.WARNING
