### Run

```shell
make up
```

For more information, see [Makefile](Makefile), [docker-compose.yml](./build/docker-compose.yaml).

File passed using docker-compose volume, so you can change it without rebuilding the image.

### Architecture

Parsing of events and processing are done in separate goroutines.

Processing results are stored in a temporary buffer, to prevent printing state before
getting validation/parsing error from the next event.

### Testing

```shell
make test
```
