"""
Base database model.
"""

# --- IMPORTS ---
from sqlalchemy import MetaData
from sqlalchemy.orm import DeclarativeBase
from sqlalchemy.orm import MappedAsDataclass


# --- GLOBALS ---
# The naming convention for constraints and indexes.
NAMING_CONVENTION = {
    'ix': 'ix_%(table_name)s_%(column_0_N_name)s',
    'uq': 'uq_%(table_name)s_%(column_0_N_name)s',
    'ck': 'ck_%(table_name)s_%(constraint_name)s',
    'fk': 'fk_%(table_name)s_%(column_0_N_name)s_%(referred_table_name)s',
    'pk': 'pk_%(table_name)s',
}


# --- CODE ---
class Base(MappedAsDataclass, DeclarativeBase):
    """
    What every model inherits from.
    """
    metadata = MetaData(naming_convention=NAMING_CONVENTION)
