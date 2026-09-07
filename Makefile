.PHONY: check validate

check: validate
	python3 -m compileall -q scripts

validate:
	python3 scripts/validate_components.py infrastructure/components.json
