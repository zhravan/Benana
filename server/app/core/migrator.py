from __future__ import annotations

from __future__ import annotations

import hashlib
from pathlib import Path
from typing import Iterable

from sqlalchemy import text, select

from .db import engine, get_session
from .models import Base, PluginMigration


def bootstrap() -> None:
    """Create host tables if not present."""
    assert engine is not None, "DB engine not initialized"
    Base.metadata.create_all(bind=engine)


def apply_sql_batch(sql: str) -> None:
    assert engine is not None
    with engine.begin() as conn:
        conn.execute(text(sql))


def _checksum(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def _list_sql_files(path: Path) -> list[Path]:
    if not path.exists() or not path.is_dir():
        return []
    return sorted([p for p in path.iterdir() if p.suffix.lower() == ".sql" and p.is_file()])


def get_applied_migrations(plugin_name: str) -> dict[str, str]:
    """Return mapping of migration_id -> checksum for a plugin."""
    with get_session() as s:
        rows: Iterable[PluginMigration] = s.execute(
            select(PluginMigration).where(PluginMigration.plugin == plugin_name)
        ).scalars()
        return {r.migration_id: r.checksum for r in rows}


def apply_plugin_migrations(plugin_name: str, migrations_dir: Path) -> None:
    """Apply lexicographically ordered .sql migrations for a plugin.

    Records each applied migration with a checksum and fails fast on drift.
    """
    files = _list_sql_files(migrations_dir)
    if not files:
        return

    applied = get_applied_migrations(plugin_name)
    for f in files:
        mig_id = f.stem
        data = f.read_bytes()
        cs = _checksum(data)
        prev = applied.get(mig_id)
        if prev is not None:
            if prev != cs:
                raise RuntimeError(
                    f"Checksum mismatch for {plugin_name}:{mig_id} (drift detected)"
                )
            # already applied with same checksum
            continue

        sql = data.decode("utf-8")
        assert engine is not None
        with engine.begin() as conn:
            conn.execute(text(sql))
            with get_session() as s:
                s.add(
                    PluginMigration(
                        plugin=plugin_name, migration_id=mig_id, checksum=cs
                    )
                )
