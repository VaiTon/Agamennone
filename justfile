compose := "docker compose"

default: achille agamennone
    @echo "🚀 Build complete!"

env:
    cp .env.example .env

services-up *ARGS:
    {{compose}} up -d {{ARGS}}

services-down *ARGS:
    {{compose}} down {{ARGS}}

services-clean:
    {{compose}} down --volumes

_build folder:
    go build ./cmd/{{folder}}/

agamennone: (_build "agamennone")
achille: (_build "achille")

up: services-up agamennone
    @echo "Starting the server..."
    ./agamennone

install:
    go install ./cmd/agamennone
    go install ./cmd/achille
    @echo "🚀 Installed the server and client binaries"

escape_analysis:
    go build -o /dev/null -gcflags '-m -l' ./...

setup-cgroups:
    #!/bin/bash
    set -euo pipefail
    if [ ! -f ~/.config/systemd/user/exploits.slice ]; then
        echo "CGroups slice not found, creating..."
        mkdir -p ~/.config/systemd/user
        echo << EOF | tee ~/.config/systemd/user/exploits.slice
        [Slice]
        CPUQuota=30%
    EOF
        systemctl --user daemon-reload
    fi

    systemctl --user start exploits.slice

    echo "🛠️  Cgroups slice setup complete!"


clean-cgroups:
    #!/bin/bash
    set -euo pipefail

    if [ -f "~/.config/systemd/user/exploits.slice" ]; then
        systemctl --user stop exploits.slice
        rm "~/.config/systemd/user/exploits.slice"
        systemctl --user daemon-reload
    fi

    echo "🧹 Cleaned up cgroups slice"


exploit *ARGS: achille
    systemd-run --user --pty \
        --slice=exploits.slice \
        --property=CPUQuota=30% \
        --working-directory="$(pwd)" \
        ./achille {{ARGS}}
