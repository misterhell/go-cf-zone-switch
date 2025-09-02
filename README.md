# Proxy ip changer

## Running the app

### Build

It will build into `bin/app`

```sh
make build
```

### Build and run

It will build into `bin/app` and start app

```sh
make start
```

### Run with special config

```sh
make build && \
    ./bin/app --config-path $(pwd)/any-config.toml
```

### Running locally

```sh
go run ./cmd/app --config-path $(pwd)/config.toml
```

## Db

App uses bolt db which create small local KV storage in changer.boltdb file
