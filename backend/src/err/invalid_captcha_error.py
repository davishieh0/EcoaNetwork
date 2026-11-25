"""
Invalid captcha error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class InvalidCaptchaError(EcoaError):
    """
    Raised when the proof of work is missing, wrong, expired or used before.
    """

    MESSAGE = 'The check did not pass. Try it again.'
    STATUS_CODE = 400
    LOG_LEVEL = logging.INFO
