"""
Tests.
"""

# --- IMPORTS ---
import os


# --- GLOBALS ---
os.environ.setdefault(
    'DATABASE_URL', 'postgresql+asyncpg://test:test@localhost:1/test'
)
os.environ.setdefault('AWS_REGION', 'sa-east-1')
os.environ.setdefault('AWS_ENDPOINT_URL', '')
os.environ.setdefault('COGNITO_USER_POOL_ID', 'sa-east-1_testpool')
os.environ.setdefault('COGNITO_CLIENT_ID', 'test-client-id')
os.environ.setdefault('COGNITO_CLIENT_SECRET', 'test-client-secret')
