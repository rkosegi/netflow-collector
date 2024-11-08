# Netflow exporter

Have you ever wondered where is your internet traffic going?

![traffic destination by country](docs/traffic_destination_by_country.png)

![traffic source by AS](docs/traffic_source_by_as.png)

## How it works

Simply put, it uses netflow protocol, specifically it uses V5 version (IPv4 only) for simplicity.
In order for your setup to work, you will either need [nfdump](https://github.com/phaag/nfdump)
or dedicated hardware such as [Mikrotik RB941](https://mikrotik.com/product/RB941-2nD)
Flows are then fed into collector that aggregates them as metrics.
Geolocation info is gathered from Maxmind GeoIP Lite database.
Necessary files can be obtained on RHEL OS (or similar) with `sudo dnf install geolite2-country geolite2-asn`


Example configuration for routerboard
```
/ip traffic-flow
set enabled=yes interfaces=wan,bridge1
/ip traffic-flow target
add dst-address=192.168.0.10 port=30000 version=5
```

_Note `192.168.0.10` is address of machine where exporter is running_

## Configurable metrics

Flows are aggregated into metrics in fully configurable manner.

Example metric
```yaml
  - name: traffic_detail
    description: Traffic detail
    labels:
      - name: sampler
        value: sampler
        converter: ipv4
      - name: protocol
        value: proto_name
        converter: str
      - name: source_country
        value: source_country
        converter: str
      - name: destination_asn_org
        value: destination_asn_org
        converter: str
```

Full example can be found [here](docs/config.yaml)

## Supported enrichers


- `maxmind_country`

  MaxMind GeoLite country data are used to add source and destination country (if applicable)
  - used attributes: `source_ip`, `destination_ip`
  - added attributes: `source_country`, `destination_country`
  - configuration options:
    - `mmdb_dir` - path to directory which holds [MaxMind GeoIP DB files](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data)

- `maxmind_asn`

  MaxMind GeoLite country data are used to add source and destination autonomous system (if applicable)
  - used attributes: `source_ip`, `destination_ip`
  - added attributes: `source_asn_org`, `destination_asn_org`
  - configuration options:
    - `mmdb_dir` - path to directory which holds [MaxMind GeoIP DB files](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data)

- `interface_mapper`
- `protocol_name`

- `reverse_dns`

  Does a reverse DNS lookup for IP and selects the first entry returned. `unknown` set if none found and ip_as_unknown is not enabled. Results (including missing) cached per `cache_duration`.

  - used attributes: `source_ip`, `destination_ip`
  - added attributes: `source_dns`, `destination_dns`
  - configuration options:
    - `cache_duration` - how long to cache result for. Default `1h`.
    - `tail_pihole` - useful if running on a server which is running `pihole`. If set, read from `pihole -t` to populate DNS cache. This cache will be used instead of a reverse DNS lookup if available. By tailing the PiHole log, we can see the original query before `CNAME` redirection and thus give a more interesting answer. Ensure that additional logging entries are enabled, e.g. `echo log-queries=extra | sudo tee /etc/dnsmasq.d/42-add-query-ids.conf ; pihole restartdns`
    - `lookup_local` - enable looking up local addresses. Default `false`.
    - `lookup_remote` - enable looking up remote addresses. Default `true`.
    - `ip_as_unknown` - if a reverse record is not available, uses the IP address itself rather than "unknown" string. Default `false`.

  e.g. add `reverse_dns` under `enrich:` and the following under `labels:`:

  ```yaml
  - name: source_ip
    value: source_ip
    converter: ipv4
  - name: destination_ip
    value: destination_ip
    converter: ipv4
  - name: source_dns
    value: source_dns
    converter: str
  - name: destination_dns
    value: destination_dns
    converter: str
  ```

  e.g. gather in/out statistics for hosts on the local network:
  ```yaml
  pipline:
  ...
    enrich:
      - reverse_dns
    metrics:
      ...
      items:
        - name: traffic_in
          description: Traffic in per host
          labels:
            - name: host
              value: destination_dns
              converter: str
        - name: traffic_out
          description: Traffic out per host
          labels:
            - name: host
              value: source_dns
              converter: str

  ...
  extensions:
    reverse_dns:
      lookup_local: true
      lookup_remote: false
      ip_as_unknown: true
  ```

## Run using podman/docker

```bash
podman run -ti -p 30000:30000/udp -p 30001:30001/tcp -u 1000 \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  -v /usr/share/GeoIP:/usr/share/GeoIP:ro \
  ghcr.io/rkosegi/netflow-collector:latest
```
