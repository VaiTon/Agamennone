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

client: _build_client

install:
    go install ./cmd/agamennone
    go install ./cmd/achille
    @echo "📦 Installed the server and client binaries"


escape_analysis:
    go build -o /dev/null -gcflags '-m -l' ./...


setup-cgroups:
    #!/bin/bash
    set -euo pipefail
    if [ ! -f ~/.config/systemd/user/exploits.slice ]; then
        mkdir -p ~/.config/systemd/user
        echo << EOF | tee ~/.config/systemd/user/exploits.slice
        [Slice]
        CPUQuota=30%
    EOF

        systemctl --user daemon-reload
    fi

    systemctl --user start exploits.slice

    echo "🛠️  Cgroups slice setup complete"


clean-cgroups:
    #!/bin/bash
    set -euo pipefail

    if [ -f "~/.config/systemd/user/exploits.slice" ]; then
        systemctl --user stop exploits.slice
        rm "~/.config/systemd/user/exploits.slice"
        systemctl --user daemon-reload
    fi

    echo "🧹 Cleaned up cgroups slice"



exploit *ARGS: _build_client setup-cgroups
    systemd-run --user --pty \
        --slice=exploits.slice \
        --property=CPUQuota=30% \
        --working-directory="$(pwd)" \
        ./achille {{ARGS}}
