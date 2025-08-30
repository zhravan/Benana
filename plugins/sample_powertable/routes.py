from fastapi import APIRouter


def get_router() -> APIRouter:
    r = APIRouter()

    @r.get("/hello")
    def hello():
        return {"plugin": "sample_powertable", "message": "Hello from powertable"}

    return r

