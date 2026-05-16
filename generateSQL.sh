#!/bin/zsh
sqlc generate
mockgen -destination=./internal/database/mock/querier.go -package=mockdb ./internal/database Querier