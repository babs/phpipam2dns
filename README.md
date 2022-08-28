# phpipam2dns

First goal of this project is to learn go.
Side effect is a tool that applies phpIPAM changelog to DNS via dynamic updates.

## usage

* `-once`: run only once and exits, by default it loops forever with a one second sleep in-between
* `-skip-state`: start to handle the changelogs from the first entry

## Configuration

### Environment variables
* `LOG_LEVEL` (optional): one of `panic`, `fatal`, `error`, `warn`, `warning`, `info`, `debug` or `trace`. Default: `info` (see [logrus ParseLevel](https://github.com/sirupsen/logrus/blob/master/logrus.go))

### Config file

Main config file is named `config.yml` and is expected to be in the current working directory. It contains dsn, zones and key definitions for forward and reverse updates as follow:

* `dsn`: using `username:password@tcp(ip-or-hostname)/database` format. For more information refer to mysql go driver [data source name](https://github.com/go-sql-driver/mysql/#dsn-data-source-name)
* `forward zones` and `reverse zones`: configuration for straight and reverse zones.
  Each of those entries are dictionaries where keys are zone name and values are server definitions, like so:
  * `server`: ip of the server
  * `algo`: key's algorithm (see: https://github.com/miekg/dns/blob/master/tsig.go)
  * `keyname`: the key name
  * `secret`: the shared secret as generated. See bellow [Bind configuration example](#bind-configuration-example)

#### Example of `config.yml`

```yaml
dsn: username:password@tcp(127.0.0.3)/phpipam
forward zones:
  example.com: &server-127-0-0-2-keyname
    server: 127.0.0.2
    algo: hmac-sha256
    keyname: keyname
    secret: wL8Yu+QOrf1+oijvVYgafKIyRYMvXTs6gEMd7yw5wZo=
  example.org: *server-127-0-0-2-keyname
reverse zones:
  0.16.172.in-addr.arpa: *server-127-0-0-2-keyname
  0.168.192.in-addr.arpa: *server-127-0-0-2-keyname
  1.168.192.in-addr.arpa: *server-127-0-0-2-keyname

```
## Bind configuration example


First, generate a key using `ddns-confgen -k keyname` or `tsig-keygen keyname` and include the `key "xxx" {[...]};` in your bind configuration.

Then configure the zones you want to update with the `allow-update` stanza.
example:
```
zone "example.com" in {
  type master;
  file "/etc/bind/masters/example.com";
  allow-update { key "keyname"; };
};

```
Make sure bind has write permission on the folder to write journal files and write permission on zone files for bind to rewrite it on `rndc sync` calls.

Reload bind and once configured accordingly, run `phpipam2dns`
