# Compatibility aliases; mise.toml owns all task implementations.
.PHONY: build fmt test vet test-acc testenv-up testenv-down
build fmt test vet:
	mise run $@
test-acc:
	mise run test:acc
testenv-up:
	mise run testenv:up
testenv-down:
	mise run testenv:down
