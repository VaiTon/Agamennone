
# Agamennone

Agamennone is a simple, resilient and scalable flag submission system.
It is designed to be simple but extensible.

Built with [Kotlin](https://kotlinlang.org/), [Ktor](https://ktor.io/) and 


## Development

1. Clone the repository
2. Run `./gradlew run` to start the server

## Building

The project can be built with:

- `./gradlew installDist`: creates a runnable distribution in `build/install`
- `./gradlew distZip`: same thing but as a zip file. The archive will be located in `build/distributions/`.
- `./gradlew distTar`: same thing but as a tar file. The archive will be located in `build/distributions/`.

The difference between the files with the `shadow` suffix and the ones without it is that, instead of packaging the dependencies into the
archive as they are, the shadow jar is used.


## Monitoring
To visualize flags, run a grafana instance with the JSON Datasource plugin:

```shell
docker run -d --name=grafana -p 3000:3000 -e "GF_INSTALL_PLUGINS=marcusolsson-json-datasource" grafana/grafana-oss
```