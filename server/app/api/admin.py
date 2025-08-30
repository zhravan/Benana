from __future__ import annotations

from fastapi import APIRouter, Depends, UploadFile
from fastapi import HTTPException, status
from fastapi import Request


router = APIRouter()


def _pm(request: Request):
    return request.app.state.plugin_manager


@router.get("/plugins")
def list_plugins(pm=Depends(_pm)):
    all_names = pm.discover()
    loaded = list(pm.loaded.keys())
    return {"available": all_names, "loaded": loaded}


@router.post("/plugins/{name}/enable")
def enable_plugin(name: str, pm=Depends(_pm)):
    try:
        pm.load(name)
        return {"status": "enabled", "name": name}
    except Exception as e:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(e))


@router.post("/plugins/{name}/disable")
def disable_plugin(name: str, pm=Depends(_pm)):
    pm.unload(name)
    return {"status": "disabled", "name": name}


@router.post("/plugins/{name}/reload")
def reload_plugin(name: str, pm=Depends(_pm)):
    pm.reload(name)
    return {"status": "reloaded", "name": name}


@router.post("/plugins/install")
def install_plugin(file: UploadFile, pm=Depends(_pm)):
    # TODO: implement safe extraction and validation in a later step
    raise HTTPException(status_code=501, detail="Install not implemented yet")


