# Agamennone

Agamennone is a simple, resilient and scalable flag submission system.

## Client usage

Agamennone has its own client, Achille, but it can also be used with the DestructiveFarm client [^1].

```shell
just client
./achille -h
```

or install it to your path via

```shell
just install
achille -h
```

### Niceness and CPU usage

To reduce CPU usage, consider using `nice` and `taskset` to set the process priority and CPU affinity.

```shell
# set the process niceness to 10 (lower than default) and use only CPU 0-7
nice -n 10 taskset -c 0-7 ./achille
```

### DestructiveFarm client

To use it, either use the original version or the one provided in the `client` folder.

[^1]: https://github.com/UlisseLab/DestructiveFarm/blob/main/docs/en/farm_client.md

## Server usage

You need docker compose and Go installed to run the server.

```
just env
# configure the config.json

just server
```

### Submitters

Submitters are stored in the `submitters` folder.
Select the submitter to use in the `config.json` file.

A submitter is an executable that:

- reads a list of flags from the standard input
- sends the flags to the game server
- writes the flag responses to the standard output with the response from the game server and an optional message

Currently supported submitters are:

| Name    | Language | Description                                                                                                                                                                                   |
| ------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `dummy` | Python   | returns a random response for each flag                                                                                                                                                       |
| `ccit`  | Python   | sends flags to the CyberChallenge game server. Made for the CyberChallenge.IT A/D National Contest. <br/>If you want to use it, you need to provide the team token INSIDE THE SUBMITTER FILE. |

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

## License

This project is licensed under the AGPL License - see the [LICENSE](LICENSE) file for details.

The code under the `client` folder is licensed under the MIT License - see the [client/LICENSE](client/LICENSE) file for
details.
