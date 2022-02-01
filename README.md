# [gqrss](https://github.com/qbit/gqrss)

A tool to query and display GitHub issues relating to anything you want.

## Libraries used

- [gorilla/feeds](https://github.com/gorilla/feeds) for generating RSS/Atom
  feeds.
- [suah.dev/protect](https://suah.dev/protect) for OpenBSD's
  [pledge](https://man.openbsd.org/pledge)/[unveil](https://man.openbsd.org/unveil).

`gqrss` will produce a `rss.xml` and `atom.xml` file in the directory it was
ran. It expects a GitHub authentication token to be present in the
`GH_AUTH_TOKEN` environment variable.
