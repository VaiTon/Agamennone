
To visualize flags, run a grafana instance with the JSON Datasource plugin

```shell
docker run -d --name=grafana -p 3000:3000 -e "GF_INSTALL_PLUGINS=marcusolsson-json-datasource" grafana/grafana-oss
```