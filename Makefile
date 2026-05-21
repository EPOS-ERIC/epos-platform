up: 
	tilt up

down:
	tilt down

populate:
	./packages/scripts/populatescript.sh

test:
	$(MAKE) -C services/gateway test
	$(MAKE) -C services/user-group test
	$(MAKE) -C services/metadata-lifecycle test

fmt:
	$(MAKE) -C services/gateway fmt
	$(MAKE) -C services/user-group fmt
	$(MAKE) -C services/metadata-lifecycle fmt

lint:
	$(MAKE) -C services/gateway lint
	$(MAKE) -C services/user-group lint
	$(MAKE) -C services/metadata-lifecycle lint

.PHONY: up down populate test fmt lint
