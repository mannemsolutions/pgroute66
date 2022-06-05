# pgroute66
A tool to direct routers to te correct postgres master

## The origin
As I was loking into the options that are around for Postgres HA integration, I noticed 2 things:
1: There was no real protection against split brain situations, which is a risk
2: There was connects and disconnects for every check, which has its overhead both the prostgres / linux, but also as logfile polution

The first issue could be remedied with a proper HA solution, but with protection on the router level just seems a bit safer.
the second issue might maybe be remedied with a pooler???
But it just seems like we are fixing issues at the wrong level.

Enter pgroute66. Pgroute66 can run as a service, can monitor multiple postgres instances, and has an API.
With pgroute66, a simple bash script could do a HAProxy check, or reconfigure pgbouncer, just by checking the state in the API.
pgroute itself maintains and reuses its connections, and wil only point to a master when it detects on.
When multiple are detected, pgroute66 will point to none of them.

Is it the safest option? I feel that in solutions where etcd is used for consensus, the correct master 
Writing a HAProxy check, or a pgbouncer reconfiguration tool 
 1 could be remedied with a proper 

While writing postgres software that manages objects in Postgres (like [pgfga](https://github.com/MannemSolutions/pgfga)), we needed a tool for easy integration testing.
As an integration test we just wanted to create an environment with Postgres, the tool, (and other components as required), run the tool and check the outcome in postgres.
We decided to build a tool which can run defined queries against Postgres, and check for expected results.
And thus [pgroute66](https://github.com/MannemSolutions/pgroute66) was born.

## Downloading pgroute66
The most straight forward way is to download [pgroute66](https://github.com/MannemSolutions/pgroute66) directly from the [github release page](https://github.com/MannemSolutions/pgroute66/releases).
But there are other options, like
- using the [container image from dockerhub](https://hub.docker.com/repository/docker/mannemsolutions/pgroute66/general)
- direct build from source (if you feel you must)

Please refer to [our download instructions](DOWNLOAD_AND_RUN.md) for more details on all options.

## Usage
After downloading the binary to a folder in your path, you can run pgroute66 with a command like:
```bash
pgroute66 ./pgroute66.yml
```

## Defining your config
You can define your config in a yaml document.
There currently is only one key `hosts`, which is a map of maps.
Every key is the name of the config, and every value is a map of key/value pairs with dsn config.

An example config could be:
```yaml
---
hosts:
  host1:
    host: 1.2.3.4
    port: 5432
    user: pgroute66
    password: cGEkJHcwcmQ=
  host2:
    host: 1.2.3.5
    port: 5432
    user: pgroute66
    password: cGEkJHcwcmQ=
  host3:
    host: 1.2.3.6
    port: 5432
    user: pgroute66
    password: cGEkJHcwcmQ=

bind: 127.0.0.1

port: 8443

ssl:
  cert: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2d0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktrd2dn
  key: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUQzakNDQXNZQ0NRRGZYZkhoanBCZHNEQU5CZ2txaGtpRzl

loglevel: debug

```

Please note that the password, ssl.cert and ssl.key are base64 encrypted values.

## calling the api
With the above defined config, the following API requests could be issued (curl examples):
```
curl -G https://127.0.0.1:8443/v1/primary
# which might return [] (when none is master, or when multiple are master)
# or ["host1"] (when host1 is master), or ["host2"], or ["host3"]

curl -G https://127.0.0.1:8443/v1/primaries
# which might return ["host1","host2","host3"] (when things are really bad)

curl -G https://127.0.0.1:8443/v1/standbys
# which could return ["host2", "host3"]

curl -G https://127.0.0.1:8443/v1/node/host1
# which could return ["primary"], ["standby"], or ["unavailable"]
```
