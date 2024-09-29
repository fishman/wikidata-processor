# wikidata-processor

Wikidata RDF processor

## Getting started

This project requires Go to be installed. On OS X with Homebrew you can just run `brew install go`.

Running it then should be as simple as:

```console
$ make build
$ ./bin/wikidata-processor --output=out_dir --chunksize=3000000 --language=en ~/Downloads/latest-all.ttl.gz 
```

### Testing

``make test``
