from __future__ import annotations

import importlib
import sys
from dataclasses import dataclass
from datetime import datetime
import uuid
from pathlib import Path
from typing import Dict, Optional

from fastapi import FastAPI, APIRouter

from .settings import get_settings, Settings
from .service_registry import ServiceRegistry
from .plugin_spi import BasePlugin
from .migrator import apply_plugin_migrations
from .db import get_session
from .models import Plugin as PluginModel


@dataclass
class LoadedPlugin:
    plugin: BasePlugin
    module_name: str
    router: APIRouter


class PluginManager:
    def __init__(self, settings: Optional[Settings] = None) -> None:
        self.settings = settings or get_settings()
        self.plugins_dir = Path(self.settings.plugins_dir)
        self.registry = ServiceRegistry()
        self.loaded: Dict[str, LoadedPlugin] = {}

    def startup(self, app: FastAPI) -> None:
        # Ensure 'plugins' namespace is importable
        root = Path.cwd()
        if str(root) not in sys.path:
            sys.path.insert(0, str(root))
        # Optionally auto-load core plugins later (after we have any)
        self.app = app

    def shutdown(self) -> None:
        # Unload all
        for name in list(self.loaded.keys()):
            self.unload(name)

    def discover(self) -> list[str]:
        if not self.plugins_dir.exists():
            return []
        names = []
        for p in self.plugins_dir.iterdir():
            if p.is_dir() and (p / "plugin.py").exists():
                names.append(p.name)
        return names

    def load(self, name: str) -> None:
        if name in self.loaded:
            return
        module_name = f"plugins.{name}.plugin"
        mod = importlib.import_module(module_name)
        if not hasattr(mod, "get_plugin"):
            raise RuntimeError(f"Plugin {name} missing get_plugin()")
        plugin: BasePlugin = mod.get_plugin()

        # Apply per-plugin migrations
        migrations_dir = plugin.migrations_path()
        apply_plugin_migrations(plugin.name, migrations_dir)

        # Ensure plugin record exists/updated
        with get_session() as s:
            row = s.query(PluginModel).filter_by(name=plugin.name).one_or_none()
            now = datetime.utcnow()
            if row is None:
                row = PluginModel(
                    id=str(uuid.uuid4()),
                    name=plugin.name,
                    version=plugin.version,
                    description=getattr(plugin, "description", ""),
                    is_core=getattr(plugin, "is_core", False),
                    status="active",
                    created_at=now,
                    updated_at=now,
                )
                s.add(row)
            else:
                row.version = plugin.version
                row.description = getattr(plugin, "description", row.description)
                row.is_core = getattr(plugin, "is_core", row.is_core)
                row.status = "active"
                row.updated_at = now

        # on_load
        plugin.on_load({"services": self.registry})

        # Register routes
        router = APIRouter()
        plugin.register_routes(self.app, router)
        self.app.include_router(router, prefix=f"/{name}", tags=[name])

        self.loaded[name] = LoadedPlugin(plugin=plugin, module_name=module_name, router=router)

    def unload(self, name: str) -> None:
        lp = self.loaded.get(name)
        if not lp:
            return
        try:
            lp.plugin.on_unload({"services": self.registry})
        finally:
            # Note: FastAPI does not provide a first-class API to deregister routes at runtime.
            # In practice, we keep routes but mark plugin disabled in DB; full removal would rebuild the app/router.
            # For now, keep simple: remove from loaded registry only.
            self.loaded.pop(name, None)

    def reload(self, name: str) -> None:
        lp = self.loaded.get(name)
        module_name = lp.module_name if lp else f"plugins.{name}.plugin"
        self.unload(name)
        if module_name in sys.modules:
            del sys.modules[module_name]
        self.load(name)
