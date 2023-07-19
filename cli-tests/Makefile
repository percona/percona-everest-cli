init:               ## Install dependencies
	npm ci
	npx playwright install

test:               ## Run all tests
	npx playwright test

lint:               ## Run linter typescript checks
	npm run lint:ts && npm run typecheck && echo 'Lint OK'

lint-fix:           ## Fix linter errors where possible
	npx eslint --fix .
