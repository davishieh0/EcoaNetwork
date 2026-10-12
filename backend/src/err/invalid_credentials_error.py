"""
Invalid credentials error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class InvalidCredentialsError(EcoaError):
    """
    Raised when the email and password do not match an account.
    """

    MESSAGE = 'Incorrect email or password!'
    STATUS_CODE = 401
    LOG_LEVEL = logging.WARNING
