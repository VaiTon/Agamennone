compose := `command -v podman-compose || echo docker-compose`

up:
    {{compose}} up -d

down:
    {{compose}} down

build folder:
    go build ./cmd/{{folder}}/

server: (build "agamennone")
    ./agamennone

client: (build "achille")
