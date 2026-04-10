# Agamennone

A simple, resilient and scalable flag submission system for Attack/Defense CTFs, built for [CyberChallenge.IT](https://cyberchallenge.it) and compatible A/D competitions.

**Requires:** [just](https://github.com/casey/just), Docker, Go

## Quick Start

```shell
git clone https://github.com/VaiTon/Agamennone.git
cd Agamennone
```

### Server

```shell
just agamennone # builds the server binary
just env        # generates .env — edit before continuing
```

Then, in config.json:

- Add your target teams
- Select a submitter (see [Submitters](#submitters))

> [!IMPORTANT]
> You **NEED** to configure a submitter. If you don't, you **WILL LOSE FLAGS**, as the dummy submitter will accept every flag.
>
> If a submitter is not available for your game server, you can write your own. See the [Submitters](#submitters) section for details.

Finally, start the server:

```shell
just services-up # or docker-compose up -d
```

See [Server (Agamennone)](#server-agamennone) for more details.

### Client

```shell
just achille # builds the client binary
```

Then, write your exploit script (see `client/spl_example.py` for an example) and run:

```shell
./achille -h                         # full flag reference
./achille ./exploit.py -t 20 -w 8    # 20s timeout, 8 workers
```

See [Client (Achille)](#client-achille) for more details.

## Architecture

The system has two components:

- **Achille** (client): runs your exploit in parallel across all target teams, collects captured flags, and forwards them to Agamennone. Buffers flags in memory and retries automatically if the server is unreachable.
- **Agamennone** (server): receives flags, deduplicates and persists them to a database, then submits them to the game server via a configurable submitter. Can be restarted mid-competition without losing queued flags.

```mermaid
flowchart LR
    E["Your exploit script"]
    A["Achille\n(N workers in parallel)"]
    S["Agamennone\n(stores & submits flags)"]
    B["Submitter"]
    G["Game Server"]

    E --> A --> S --> B --> G
```

## Server (Agamennone)

### Configuration

Configure the server via `config.json`. Run `just env` to generate a starting `.env` file.

### Submitters

A submitter is an executable that reads flags from stdin, submits them to the game server, and writes responses to stdout.

Submitters live in the `submitters` folder and are selected in `config.json`.

| Name    | Language | Description                                                                                                 |
| ------- | -------- | ----------------------------------------------------------------------------------------------------------- |
| `dummy` | Python   | Returns a random response for each flag. Useful for testing                                                 |
| `ccit`  | Python   | Submits to the CyberChallenge.IT game server. Requires your **team token hardcoded in the submitter file**. |

### Database

Agamennone supports MariaDB and SQLite.

#### MariaDB (recommended)

```shell
./agamennone -db "mariadb://user:password@tcp(localhost:3306)/agamennone"
```

The default `compose.yml` includes a MariaDB instance. Running `./agamennone` without arguments connects to `localhost:3306` with `agamennone:agamennone` credentials — change these in production.

#### SQLite

```shell
./agamennone -db "sqlite://agamennone.db"
```

> [!IMPORTANT]
> SQLite is not recommended for production use and does not support Grafana monitoring.

## Client (Achille)

```shell
./achille -h                              # full flag reference
./achille ./exploit.py -t 20 -w 8 -o     # 20s interval, 8 workers, one-shot
```

Achille runs your exploit in parallel across all target teams. If the server is unreachable, flags are buffered in memory and retried until successfully delivered.

### CPU Usage

For long-running competitions, consider pinning Achille to specific cores and lowering its priority:

```shell
nice -n 10 taskset -c 0-7 ./achille ./exploit.py
```

## License

The server is licensed under the [AGPL License](./LICENSE).
The client (`/client`) is licensed under the [MIT License](./client/LICENSE).
