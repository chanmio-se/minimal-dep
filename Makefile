.PHONY: build run test e2e lint quint migrate modulegen sqlc

build:
	go build ./...

run:
	go run ./cmd/server

test:
	go test ./...

e2e:
	go test -tags=e2e ./tests/e2e/...

lint:
	go tool golangci-lint custom
	./bin/custom-gcl run ./...
	go vet ./...

quint:
	@for m in session authz csrf; do \
		npx --yes @informalsystems/quint@0.34.0 parse "spec/quint/$$m.qnt" && \
		npx --yes @informalsystems/quint@0.34.0 typecheck "spec/quint/$$m.qnt" && \
		npx --yes @informalsystems/quint@0.34.0 run "spec/quint/$$m.qnt" --max-steps=10 || exit 1; \
	done

migrate:
	go run ./cmd/migrate

modulegen:
	@test -n "$(NAME)" || (echo "usage: make modulegen NAME=<name>" && exit 1)
	go run ./cmd/modulegen --name=$(NAME)

sqlc:
	sqlc generate
