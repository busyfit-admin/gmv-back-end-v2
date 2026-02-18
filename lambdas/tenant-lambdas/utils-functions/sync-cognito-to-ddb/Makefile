.PHONY: help test install clean lint coverage

help:
	@echo "Available targets:"
	@echo "  install   - Install dependencies"
	@echo "  test      - Run unit tests"
	@echo "  coverage  - Run tests with coverage report"
	@echo "  lint      - Run code linting"
	@echo "  clean     - Clean up generated files"

install:
	pip3 install -r requirements.txt
	pip3 install pytest pytest-cov pylint

build:
	@echo "No build step required for this Python module."

test:
	python3 -m unittest test_sync_cognito_to_ddb.py -v

coverage:
	python3 -m pytest test_sync_cognito_to_ddb.py --cov=sync_cognito_to_ddb --cov-report=html --cov-report=term
	@echo "Coverage report generated in htmlcov/index.html"

lint:
	pylint sync_cognito_to_ddb.py

clean:
	rm -rf __pycache__ .pytest_cache htmlcov .coverage
	find . -type f -name '*.pyc' -delete
	find . -type d -name '__pycache__' -delete
