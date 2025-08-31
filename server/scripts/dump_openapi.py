import asyncio
import json
import os

from app.main import app


async def _run():
    # Ensure startup events run so plugins autoload and routes are mounted
    await app.router.startup()
    try:
        schema = app.openapi()
        out = os.environ.get("BENANA_OPENAPI_OUT", "server/openapi.json")
        # Ensure directory exists
        os.makedirs(os.path.dirname(out), exist_ok=True)
        with open(out, "w", encoding="utf-8") as f:
            json.dump(schema, f, ensure_ascii=False, indent=2)
        print(f"OpenAPI schema written to {out}")
    finally:
        await app.router.shutdown()


if __name__ == "__main__":
    asyncio.run(_run())

