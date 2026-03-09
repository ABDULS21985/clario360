PYTHON ?= python3

.PHONY: install test lint type build

install:
	$(PYTHON) -m pip install -e ".[all]"

test:
	$(PYTHON) -m pytest

lint:
	$(PYTHON) -m ruff check clario360 clario360_cli tests

type:
	$(PYTHON) -m mypy --strict clario360 clario360_cli

build:
	$(PYTHON) -m pip install build
	$(PYTHON) -m build
