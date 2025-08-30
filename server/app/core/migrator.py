from __future__ import annotations

from __future__ import annotations

import hashlib
from pathlib import Path
from typing import Iterable, List

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


def _is_down_file(path: Path) -> bool:
    # convention: 0001_xxx.down.sql
    return path.name.endswith(".down.sql")


def _migration_id_from_path(path: Path) -> str:
    # strip .sql and optional .down suffix
    stem = path.stem  # e.g., 0001_init or 0001_init.down
    if stem.endswith(".down"):
        stem = stem[: -len(".down")]
    return stem


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
        if _is_down_file(f):
            continue  # skip down files on apply
        mig_id = _migration_id_from_path(f)
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


def list_applied(plugin_name: str) -> List[PluginMigration]:
    with get_session() as s:
        rows: Iterable[PluginMigration] = s.execute(
            select(PluginMigration).where(PluginMigration.plugin == plugin_name).order_by(PluginMigration.applied_at.desc())
        ).scalars()
        return list(rows)


def rollback_plugin_migrations(plugin_name: str, migrations_dir: Path, steps: int = 1) -> list[str]:
    """Rollback the last N applied migrations for a plugin by running corresponding .down.sql files.

    Each rollback is executed in its own transaction; on success, the migration record is removed.
    """
    if steps <= 0:
        return []
    applied_rows = list_applied(plugin_name)
    to_rollback = applied_rows[:steps]
    rolled: list[str] = []
    for row in to_rollback:
        mig_id = row.migration_id
        down_file = migrations_dir / f"{mig_id}.down.sql"
        if not down_file.exists():
            raise RuntimeError(f"Down migration not found for {plugin_name}:{mig_id} -> {down_file.name}")
        sql = down_file.read_text(encoding="utf-8")
        assert engine is not None
        with engine.begin() as conn:
            conn.execute(text(sql))
            with get_session() as s:
                # remove record
                rec = s.get(PluginMigration, row.id)
                if rec is not None:
                    s.delete(rec)
        rolled.append(mig_id)
    return rolled
