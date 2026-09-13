ifneq (,$(wildcard .env))
    include .env
    export
endif

db-up:
	podman compose up -d

up:
	go run cmd/api/main.go

build:
	go build cmd/api/main.go

migrate:
	dbmate -d ./sql/migrations/ -u "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@127.0.0.1:5432/$(POSTGRES_DB)?sslmode=disable" status

migrate-up:
	dbmate -d ./sql/migrations/ -u "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@127.0.0.1:5432/$(POSTGRES_DB)?sslmode=disable" up
