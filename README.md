# iprepd

iprepd is a centralized IP reputation daemon that can be used to store reputation information
for IP addresses and retrieve reputation scores for addresses.

The daemon provides an HTTP API for requests, and uses a Redis server as the backend storage
mechanism. Multiple instances of the daemon can be deployed using the same Redis backend.

## Configuration

Configuration is done through the configuration file, by default `./iprepd.yaml`. The location
can be overridden with the `-c` flag.

See [iprepd.yaml.sample](./iprepd.yaml.sample) for an example configuration.

## Building the Docker image

```bash
./write_version_json.sh
docker build -t iprepd:latest .
```

Docker images are also [published](https://hub.docker.com/r/mozilla/iprepd/).

```bash
docker pull mozilla/iprepd:latest
docker run -ti --rm -v `pwd`/iprepd.yaml:/app/iprepd.yaml mozilla/iprepd:latest
```

## API

#### GET /10.0.0.1

Request the reputation for an IP address. Responds with 200 and a JSON document describing the
reputation if found. Responds with a 404 if the IP address is unknown to iprepd, or is in the
exceptions list.

The response body may include a `decayafter` element if the reputation for the address was changed
with a recovery suppression applied. If the timestamp is present, it indicates the time after which
the reputation for the address will begin to recover.

##### Response body

```json
{
	"ip": "10.0.0.1",
	"reputation": 75,
	"reviewed": false,
	"lastupdated": "2018-04-23T18:25:43.511Z"
}
```

#### DELETE /10.0.0.1

Deletes the reputation entry for the IP address.

#### PUT /10.0.0.1

Sets a reputation score for the IP address. A reputation JSON document must be provided with the
request body. The `reputation` field must be provided in the document. The reviewed field
can be included and set to true to toggle the reviewed field for a given reputation entry.

Note that if the reputation decays back to 100, if the reviewed field is set on the entry it will
toggle back to false.

The reputation will begin to decay back to 100 immediately for the address based on the decay
settings in the configuration file. If it is desired that the reputation should not decay for a
period of time, the `decayafter` field can be set with a timestamp to indicate when the reputation
decay logic should begin to be applied for the entry.

##### Request body

```json
{
	"ip": "10.0.0.1",
	"reputation": 75
}
```

#### GET /violations

Returns violations configured in iprepd in a JSON document.

##### Response body

```json
[
	{"name": "violation1", "penalty": 5, "decreaselimit": 50},
	{"name": "violation2", "penalty": 25, "decreaselimit": 0},
]
```

#### PUT /violations/10.0.0.1

Applies a violation penalty to an IP address.

If an unknown violation penalty is submitted, this endpoint will still return 200, but the
error will be logged.

If desired, `suppress_recovery` can be included in the request body and set to an integer which
indicates the number of seconds that must elapse before the reputation for this entry will begin
to decay back to 100. If this setting is not included, the reputation will begin to decay
immediately. If the violation is being applied to an existing entry, the `suppress_recovery` field
will only be applied if the existing entry has no current recovery suppression, or the specified
recovery suppression time frame would result in a time in the future beyond which the entry
currently has. If `suppress_recovery` is included it must be less than `259200` (72 hours).

##### Request body

```json
{
	"ip": "10.0.0.1",
	"violation": "violation1"
}
```

#### PUT /violations

Applies a violation penalty to a multiple IP addresses.

If an unknown violation penalty is submitted, this endpoint will still return 200, but the
error will be logged.

##### Request body

```json
[
	{"ip": "10.0.0.1", "violation": "violation1"},
	{"ip": "10.0.0.2", "violation": "violation1"},
	{"ip": "10.0.0.3", "violation": "violation2"}
]
```

#### GET /\_\_heartbeat\_\_

Service heartbeat endpoint.

#### GET /\_\_lbheartbeat\_\_

Service heartbeat endpoint.

#### GET /\_\_version\_\_

Return version data.

## Acknowledgements

The API design and overall concept for this project are based on work done in
[Tigerblood](https://github.com/mozilla-services/tigerblood).
