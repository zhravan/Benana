.PHONY: poetry-install dev run fmt lint test migrate

poetry-install:
	poetry install --no-root

dev:
	poetry run uvicorn app.main:app --reload --host 0.0.0.0 --port 8000

run:
	poetry run uvicorn app.main:app --host 0.0.0.0 --port 8000

fmt:
	poetry run black app
	poetry run isort app

lint:
	poetry run mypy app || true

test:
	echo "no tests yet"

migrate:
	poetry run python -c "from app.core.migrator import bootstrap; bootstrap()"

