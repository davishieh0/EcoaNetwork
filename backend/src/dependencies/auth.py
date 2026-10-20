"""
Dependency providers for authentication.
"""

# --- IMPORTS ---
from fastapi import Request
from src.clients.tokens import TokenVerifier
from src.err.not_authenticated_error import NotAuthenticatedError
from src.schemas.auth import AccessClaims


# --- GLOBALS ---
# the cookie the access token travels in
ACCESS_TOKEN_COOKIE = 'access_token'


# --- CODE ---
async def get_current_account(request: Request) -> AccessClaims:
    """
    Provides who is making the request, from their access token.

    :param request: FastAPI request object.

    :raises NotAuthenticatedError: If there is no valid access token.
    :raises AccessTokenExpiredError: If the access token has expired.

    :return: The caller's verified claims.
    """
    token = request.cookies.get(ACCESS_TOKEN_COOKIE)

    # signed out, or the cookie never reached us
    if not token:
        raise NotAuthenticatedError('no access token cookie')

    tokens: TokenVerifier = request.app.state.tokens

    return await tokens.verify(token)
