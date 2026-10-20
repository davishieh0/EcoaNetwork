"""
Email not verified error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class EmailNotVerifiedError(EcoaError):
    """
    Raised when an account signs in before confirming its email.
    """

    MESSAGE = 'Email not verified!'
    STATUS_CODE = 403
    LOG_LEVEL = logging.WARNING
