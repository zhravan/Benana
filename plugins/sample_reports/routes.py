from fastapi import APIRouter


def get_router() -> APIRouter:
    r = APIRouter()

    @r.get("/summary")
    def summary():
        return {"plugin": "sample_reports", "summary": {"users": 0, "reports": 0}}

    return r

