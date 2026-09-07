.PHONY: check validate go-test go-build container-build

check: validate go-test go-build
	python3 -m compileall -q scripts

validate:
	python3 scripts/validate_components.py infrastructure/components.json

go-test:
	cd submit && go test ./...

go-build:
	cd submit && CGO_ENABLED=0 go build ./cmd/amiguard-submit

container-build:
	podman build -t localhost/amiguard-submit:local -f submit/Containerfile submit
