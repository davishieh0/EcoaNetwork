"""
Dependency providers for authentication.
"""

# --- IMPORTS ---
from fastapi import Depends
from fastapi import Request
from src.clients.tokens import TokenVerifier
from src.err.not_authenticated_error import NotAuthenticatedError
from src.schemas.auth import AccessClaims
from src.utils.cookies import ACCESS_TOKEN_COOKIE


# --- CODE ---
def get_tokens(request: Request) -> TokenVerifier:
    """
    Provides the access token verifier set up at startup.

    :param request: FastAPI request object.

    :return: The application's token verifier.
    """
    tokens: TokenVerifier = request.app.state.tokens

    return tokens


async def get_current_account(
    request: Request,
    tokens: TokenVerifier = Depends(get_tokens),
) -> AccessClaims:
    """
    Provides who is making the request, from their access token.

    :param request: FastAPI request object.
    :param tokens: The access token verifier.

    :raises NotAuthenticatedError: If there is no valid access token.
    :raises AccessTokenExpiredError: If the access token has expired.

    :return: The caller's verified claims.
    """
    token = request.cookies.get(ACCESS_TOKEN_COOKIE)

    # signed out, or the cookie never reached us
    if not token:
        raise NotAuthenticatedError('no access token cookie')

    return await tokens.verify(token)
