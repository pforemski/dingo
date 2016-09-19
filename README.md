# dingo

A DNS client in Go that supports the [Google DNS-over-HTTPS](https://developers.google.com/speed/public-dns/docs/dns-over-https).
It effectively encrypts all your DNS traffic.

The ultimate goal for the project is to provide a secure, caching DNS proxy that communicates with recursive DNS resolvers over encrypted channels only. For now, it resolves DNS queries over HTTPS/1.1, in a few independent threads. The plans for future plans include HTTP/2.0 and QUIC support, better caching, and other resolvers (e.g. [OpenResolve](https://www.openresolve.com/) by OpenDNS).

# How to use it?

Download a pre-built binary for your platform from [the latest release](https://github.com/pforemski/dingo/releases/latest) (or build your own).

Run dingo as root on port 53. For example, on Linux:
```
$ sudo ./dingo-linux-amd64 -port=53
```

Update your DNS configuration. On Linux, update your `/etc/resolv.conf` file as root (remember to make backup first):
```
$ sudo sh -c "echo nameserver 127.0.0.1 > /etc/resolv.conf"
```
