
## Developing

Simply have a modern Go version installed; check out the project somewhere (NOT in $GOPATH, as we use Go Modules),
and then run `./build.sh`.

## Releasing new versions


### Prerequisites for releasing

1. ensure you have [goreleaser](https://goreleaser.com/) installed:

  ```bash
  brew install goreleaser/tap/goreleaser
  ```

2. Create a new token for goreleaser [in your GitHub settings](https://github.com/settings/tokens); select the `repo` scope.

3. put the just-created token into the file `~/.config/goreleaser/github_token`



### Doing the release

Testing a release:

```
goreleaser --snapshot --skip-publish --rm-dist --debug
```

Executing a release:

1. Commit all changes, create a new tag and push it.

```
TAG=v0.9.0 git tag $TAG; git push origin $TAG
```

2. run goreleaser:

```
goreleaser --rm-dist
```

## Inspiration and resources

- [Docker CLI Plugins 1](https://gist.github.com/thaJeztah/b7950186212a49e91a806689e66b317d)
- [Docker CLI Plugins 2](https://dille.name/slides/2019-06-06/020_advanced/080_docker_cli_plugins/slides/)