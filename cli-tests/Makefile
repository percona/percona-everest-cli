init:               ## Install dependencies
	npm ci
	npx playwright install

install-operators:  ## Install operators to k8s
	../bin/everest install operators \
		--skip-wizard \
		--backup.enable=0 \
		--monitoring.enable=0

test-cli:           ## Run all tests
	npx playwright test --project=cli

test-install-operators:       ## Run install operators tests. Requires minikube installed and /etc/hosts updated
	npx playwright test --project=install-operators

lint:               ## Run linter typescript checks
	npm run lint:ts && npm run typecheck && echo 'Lint OK'

lint-fix:           ## Fix linter errors where possible
	npx eslint --fix .
