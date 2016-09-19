# dingo

A caching DNS proxy for the [Google DNS-over-HTTPS](https://developers.google.com/speed/public-dns/docs/dns-over-https).
It effectively encrypts all your DNS traffic.

For now, it resolves DNS queries over HTTPS/1.1, in a few independent threads (configurable).
Future plans include HTTP/2.0 and QUIC support, better caching, and support for other resolvers.

You can start it as root using:
```
root@localhost:~# go run ./dingo.go ./gdns.go -port=53
```

Remember to prepare your Go environment and download all dependencies first.

Alternatively, use pre-built binaries in the `./release` sub-directory.

Note that you will need to update your `/etc/resolv.conf` file to use `dingo` as your system-wide resolver.
