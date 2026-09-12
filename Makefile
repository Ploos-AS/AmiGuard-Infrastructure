.PHONY: check validate hardening qualification-records inventory-test go-test go-build container-build rootless-qualify

check: validate hardening qualification-records inventory-test go-test go-build
	python3 -m compileall -q scripts

validate:
	python3 scripts/validate_components.py infrastructure/components.json

hardening:
	python3 scripts/validate_m2_hardening.py
	python3 scripts/validate_m4_closed_vps_contract.py
	python3 scripts/validate_m6_5_deploy_contract.py

qualification-records:
	python3 scripts/validate_m6_8_live_qualification.py

inventory-test:
	cd scripts && python3 test_inventory_quarantine.py

go-test:
	cd submit && go test ./...

go-build:
	cd submit && CGO_ENABLED=0 go build ./cmd/amiguard-submit
	cd submit && CGO_ENABLED=0 go build ./cmd/amiguard-admin

container-build:
	podman build -t localhost/amiguard-submit:local -f submit/Containerfile submit

rootless-qualify:
	sh scripts/qualify_rootless_podman.sh
