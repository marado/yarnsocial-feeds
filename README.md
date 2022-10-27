# feeds

`feeds` is an RSS/Atom feed aggregator for [twtxt](https://twtxt.readthedocs.io/en/latest/)
that consumes RSS/Atom feeds and processes them into twtxt feeds. These can
then be consumed by any standard twtxt client such as:

- [twtxt](https://github.com/buckket/twtxt)
- [twet](https://github.com/quite/twet)
- [txtnish](https://github.com/mdom/txtnish)
- [twtxtc](https://github.com/neauoire/twtxtc)

There is also a publically (_free_) service online available at:

- https://feeds.twtxt.net/

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
