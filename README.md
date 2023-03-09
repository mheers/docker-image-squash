# docker-image-squash

> A simple tool to to squash all layers of a docker image into a single tar file.

## Dependencies
- docker

## Installation

### Binary
```bash
go install github.com/mheers/docker-image-squash/cmd@latest
```

## Usage
```bash
docker-image-squash <image> <output.tar>
```

## TODO
- [ ] remove dependency of `docker`


## Alternatives

- [docker-squash](https://github.com/goldmann/docker-squash) - written in python
- [docker-squash](https://github.com/jwilder/docker-squash) - written in go (not maintained)


### Why docker-image-squash?

- no need to install python
- no need to run as root
