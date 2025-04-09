compose := "docker compose"

default: _build_client _build_server
    @echo "🚀 Build complete!"

env:
    cp .env.example .env

up:
    {{compose}} up -d

down *ARGS:
    {{compose}} down {{ARGS}}

clean:
    {{compose}} down --volumes

build folder:
    go build ./cmd/{{folder}}/


_build_server: (build "agamennone")
_build_client: (build "achille")

server: (build "agamennone") up
    ./agamennone

client: (build "achille")

install:
    go install ./cmd/agamennone
    go install ./cmd/achille
    @echo "📦 Installed the server and client binaries"


escape_analysis:
    go build -o /dev/null -gcflags '-m -l' ./...
