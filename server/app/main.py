from fastapi import FastAPI
from fastapi.responses import ORJSONResponse

from .core.settings import get_settings
from .core.db import init_engine_and_session
from .core.migrator import bootstrap
from .core.plugin_manager import PluginManager
from .api.admin import router as admin_router


app = FastAPI(default_response_class=ORJSONResponse, title="Benana Host Runtime")


@app.on_event("startup")
def on_startup():
    settings = get_settings()

    # Initialize DB engine/session
    init_engine_and_session(settings)

    # Ensure host bootstrap tables exist
    bootstrap()

    # Initialize plugin manager and load active/core plugins later
    app.state.plugin_manager = PluginManager(settings=settings)
    app.state.plugin_manager.startup(app)


@app.on_event("shutdown")
def on_shutdown():
    pm: PluginManager = app.state.plugin_manager
    pm.shutdown()


@app.get("/health")
def health():
    return {"status": "ok"}


app.include_router(admin_router, prefix="/admin", tags=["admin"])

