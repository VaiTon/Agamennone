compose := `command -v podman-compose || echo docker-compose`

up:
    {{compose}} up -d

down:
    {{compose}} down

build:
    go build ./cmd/agamennone/

run:
    go run ./cmd/agamennone/



