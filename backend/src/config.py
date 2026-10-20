"""
Application configuration.
"""

# --- IMPORTS ---
from pydantic import SecretStr
from pydantic_settings import BaseSettings
from pydantic_settings import SettingsConfigDict


# --- CONFIG ---
class Config(BaseSettings):
    """
    Reads configuration from environment variables and .env file.
    """

    # general
    ENVIRONMENT: str = 'development'
    LOG_LEVEL: str = 'INFO'

    # routing
    API_PREFIX: str = '/api'

    # cors
    CORS_ORIGINS: str = 'http://localhost:*'

    # database
    DATABASE_URL: SecretStr

    # aws (the endpoint is only set when pointing at LocalStack)
    AWS_REGION: str
    AWS_ENDPOINT_URL: str = ''

    # cognito (the issuer is only set when pointing at LocalStack)
    COGNITO_USER_POOL_ID: str
    COGNITO_CLIENT_ID: str
    COGNITO_CLIENT_SECRET: SecretStr
    COGNITO_ISSUER: str = ''

    # cookies
    COOKIE_SECURE: bool = True

    @property
    def cors_origins(self) -> list[str]:
        """
        Parses CORS_ORIGINS into a list.

        :return: List of allowed CORS origins.
        """
        return [o.strip() for o in self.CORS_ORIGINS.split(',') if o.strip()]


    @property
    def is_production(self) -> bool:
        """
        Whether this instance runs in production.

        :return: True when ENVIRONMENT is 'production'.
        """
        return self.ENVIRONMENT.strip().lower() == 'production'


    @property
    def cognito_issuer(self) -> str:
        """
        The issuer every token from the user pool must carry.

        :return: COGNITO_ISSUER when set, otherwise the pool's AWS issuer.
        """
        if self.COGNITO_ISSUER.strip():
            return self.COGNITO_ISSUER.strip()

        return (
            f'https://cognito-idp.{self.AWS_REGION}.amazonaws.com/'
            f'{self.COGNITO_USER_POOL_ID}'
        )


    @property
    def cognito_jwks_uri(self) -> str:
        """
        Where the user pool publishes the keys that sign its tokens.

        :return: The JWKS URL under the issuer.
        """
        return f'{self.cognito_issuer}/.well-known/jwks.json'

    # pydantic settings
    model_config = SettingsConfigDict(env_file='.env', extra='ignore')


# shared singleton: import `config` instead of instantiating Config again
config = Config()
