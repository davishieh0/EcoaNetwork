"""
Session cookies.
"""

# --- IMPORTS ---
from fastapi import Response
from src.config import config
from src.schemas.auth import Tokens


# --- GLOBALS ---
ACCESS_TOKEN_COOKIE = 'access_token'
REFRESH_TOKEN_COOKIE = 'refresh_token'
SESSION_MAX_AGE = 30 * 24 * 60 * 60
ACCESS_TOKEN_PATH = config.API_PREFIX
REFRESH_TOKEN_PATH = f'{config.API_PREFIX}/auth'


# --- CODE ---
def _set(response: Response, name: str, value: str, path: str) -> None:
    """
    Sets one session cookie, out of reach of the page's JavaScript.

    :param response: The response that carries the cookie.
    :param name: The cookie's name.
    :param value: The token it holds.
    :param path: The routes the browser sends it to.

    :return: None.
    """
    response.set_cookie(
        name,
        value,
        max_age=SESSION_MAX_AGE,
        path=path,
        httponly=True,
        secure=config.COOKIE_SECURE,
        samesite='lax',
    )


def set_session_cookies(response: Response, tokens: Tokens) -> None:
    """
    Stores a session's tokens in the browser.

    :param response: The response that carries the cookies.
    :param tokens: What sign-in or refresh handed back. A refresh hands back
        no refresh token, and the one the browser has is kept.

    :return: None.
    """
    _set(response, ACCESS_TOKEN_COOKIE, tokens.access_token, ACCESS_TOKEN_PATH)

    if tokens.refresh_token:
        _set(
            response,
            REFRESH_TOKEN_COOKIE,
            tokens.refresh_token,
            REFRESH_TOKEN_PATH,
        )


def clear_session_cookies(response: Response) -> None:
    """
    Removes a session's tokens from the browser.

    :param response: The response that carries the removal.

    :return: None.
    """
    for name, path in (
        (ACCESS_TOKEN_COOKIE, ACCESS_TOKEN_PATH),
        (REFRESH_TOKEN_COOKIE, REFRESH_TOKEN_PATH),
    ):
        response.delete_cookie(
            name,
            path=path,
            httponly=True,
            secure=config.COOKIE_SECURE,
            samesite='lax',
        )
