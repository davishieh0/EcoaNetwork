"""
Access token verification.
"""

# --- IMPORTS ---
from datetime import UTC
from datetime import datetime
from jwt import ExpiredSignatureError
from jwt import InvalidTokenError
from jwt import PyJWKClient
from jwt import PyJWKClientConnectionError
from jwt import PyJWKClientError
from src.err.access_token_expired_error import AccessTokenExpiredError
from src.err.auth_provider_error import AuthProviderError
from src.err.not_authenticated_error import NotAuthenticatedError
from src.schemas.auth import AccessClaims
from typing import Any

import asyncio
import jwt


# --- GLOBALS ---
ALGORITHMS = ['RS256']
REQUIRED_CLAIMS = ['exp', 'iat', 'iss', 'sub', 'token_use', 'client_id']
KEYS_LIFESPAN = 3600
FETCH_TIMEOUT = 5


# --- CODE ---
class TokenVerifier:
    """
    Checks the access tokens the user pool issues, without calling it.
    """

    def __init__(self, issuer: str, jwks_uri: str, client_id: str) -> None:
        """
        Initialize the verifier.

        :param issuer: The issuer every token must carry.
        :param jwks_uri: Where the user pool publishes its keys.
        :param client_id: The app client every token must be issued to.

        :return: None.
        """
        self._issuer = issuer
        self._client_id = client_id
        self._keys = PyJWKClient(
            jwks_uri,
            cache_keys=True,
            lifespan=KEYS_LIFESPAN,
            timeout=FETCH_TIMEOUT,
        )


    async def verify(
        self,
        token: str,
        *,
        allow_expired: bool = False,
    ) -> AccessClaims:
        """
        Checks an access token and reads who it belongs to.

        :param token: The access token.
        :param allow_expired: Accept a token past its expiry, for the one
            case that needs it: reading who is refreshing their session.

        :raises AccessTokenExpiredError: If the token is genuine but expired.
        :raises NotAuthenticatedError: If it is not a token this pool issued
            to this app, or it was tampered with.
        :raises AuthProviderError: If the published keys cannot be fetched.

        :return: The token's claims.
        """
        # get the key that signed it
        try:
            key = await asyncio.to_thread(
                self._keys.get_signing_key_from_jwt, token
            )

        # the keys could not be fetched: the token may well be fine
        except PyJWKClientConnectionError as error:
            raise AuthProviderError(f'fetching the keys: {error}') from error

        # no published key signed it, or it is not a token at all
        except (PyJWKClientError, InvalidTokenError) as error:
            raise NotAuthenticatedError(str(error)) from error

        # read the claims
        try:
            claims: dict[str, Any] = jwt.decode(
                token,
                key,
                algorithms=ALGORITHMS,
                issuer=self._issuer,
                options={
                    'require': REQUIRED_CLAIMS,
                    'verify_exp': not allow_expired,
                },
            )

        # the token is genuine but expired, and we are not allowing that
        except ExpiredSignatureError as error:
            raise AccessTokenExpiredError(str(error)) from error

        # bad signature, wrong issuer, a required claim missing
        except InvalidTokenError as error:
            raise NotAuthenticatedError(str(error)) from error

        # token is not an access token: it is a refresh token or an ID token
        if claims['token_use'] != 'access':
            raise NotAuthenticatedError(f'token_use is {claims["token_use"]}')

        # the token was issued to another app client: it is not ours
        if claims['client_id'] != self._client_id:
            raise NotAuthenticatedError('issued to another app client')

        # return the claims in a typed object
        return AccessClaims(
            sub=claims['sub'],
            expires_at=datetime.fromtimestamp(claims['exp'], UTC),
        )
