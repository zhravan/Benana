from __future__ import annotations

from sqlalchemy import text

from .db import engine
from .models import Base


def bootstrap() -> None:
    """Create host tables if not present."""
    assert engine is not None, "DB engine not initialized"
    Base.metadata.create_all(bind=engine)


def apply_sql_batch(sql: str) -> None:
    assert engine is not None
    with engine.begin() as conn:
        conn.execute(text(sql))

