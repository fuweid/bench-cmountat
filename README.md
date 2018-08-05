# fork-mountat + reexec-mountat benchmarks

It runs benchmarks for fork-mountat(aka `fmountat`) and reexec-mountat(
aka `rmountat`).

## Usage

The benchmark case run with root privilege.

```
$ sudo env "PATH=$PATH" go test -v -bench=.
```

> NOTE: `env "PATH=$PATH"` uses current environment to make sure `go` command
is available for `root`.
