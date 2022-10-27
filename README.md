# feeds

`feeds` is an RSS/Atom feed aggregator for [twtxt](https://twtxt.readthedocs.io/en/latest/)
that consumes RSS/Atom feeds and processes them into twtxt feeds. These can
then be consumed by any standard twtxt client such as:

- [twtxt](https://github.com/buckket/twtxt)
- [twet](https://github.com/quite/twet)
- [txtnish](https://github.com/mdom/txtnish)
- [twtxtc](https://github.com/neauoire/twtxtc)

`feeds` is also used as a "Feed Source" for [Yarn.social](https://yarn.social)
which you can fund under the "Feeds" page, for example https://twtxt.net/feeds
(_you must be logged in_).

There is also a publically (_free_) service online available at:

- https://feeds.twtxt.net/

The [feeds.twtxt.net](https://feeds.twtxt.net) instance is also the default
"Feed Source" for all [Yarnsocial](https://yarn.social) pods running `yarnd`
([yarn](https://git.mills.io/yarnsocial/yarn)) -- which can be configured with
`--feed-sources`:

```console
$ yarnd --help 2>&1 | grep feed-sources
      --feed-sources strings          external feed sources for discovery of other feeds (default [https://feeds.twtxt.net/we-are-feeds.txt])
```

![Screenshot 1](./screenshot1.png)
![Screenshot 2](./screenshot2.png)

## Installation

### Source

```#!bash
$ go get -u git.mills.io/yarnsocial/feeds
```

## Usage

Run `feeds`:

```#!bash
$ feeds
```

Then visit: http://localhost:8000/

## Related Projects

- [Yarn](https://git.mills.io/yarnsocial/yarn)

## License

`feeds` is licensed under the terms of the [MIT License](/LICENSE)
