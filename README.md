# Agamennone

Agamennone is a simple, resilient and scalable flag submission system.

## Usage

1. Clone the repository
2. Run `go build ./cmd/agamennone` to build the project
3. Run `docker-compose up` or `podman-compose up` to start the database and the Grafana instance
4. Configure the server in the `config.json` file
5. Run `./agamennone` to start the server

### Submitters

Submitters are stored in the `submitters` folder.
Select the submitter to use in the `config.json` file.

A submitter is an executable that:

- reads a list of flags from the standard input
- sends the flags to the game server
- writes the flag responses to the standard output with the response from the game server and an optional message

Currently supported submitters are:

| Name    | Language | Description                                                                                                                                                                                   |
|---------|----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `dummy` | Python   | returns a random response for each flag                                                                                                                                                       |
| `ccit`  | Python   | sends flags to the CyberChallenge game server. Made for the CyberChallenge.IT A/D National Contest. <br/>If you want to use it, you need to provide the team token INSIDE THE SUBMITTER FILE. |

### Clients

Agamennone is compatible with the DestructiveFarm client [^1].
To use it, follow the instructions in the client repository and configure the client to uses the Agamennone HTTP API.

[^1]: https://github.com/UlisseLab/DestructiveFarm/blob/main/docs/en/farm_client.md

## Storage

Agamennone supports two storage backends, mariadb and sqlite.

### MariaDB

To use mariadb, start the database with the `-db` argument set to the mariadb connection string prefixed
with `mariadb://`:

```shell
./agamennone -db "mariadb://user:password@tcp(localhost:3306)/agamennone"
# or just
./agamennone
# if the database is running on localhost:3306 with agamennone:agamennone credentials and agamennone database
```

The compose.yml setup includes a mariadb instance.

### SQLite

To use sqlite, start the database with the `-db` argument set to the sqlite connection string prefixed with `sqlite://`:

```shell
./agamennone -db "sqlite://agamennone.db"
```

> [!IMPORTANT]
> SQLite is not recommended for production use.
>
> It can't be used with Grafana, so no monitoring is available.