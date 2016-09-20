# dingo

A DNS client in Go that supports the [Google
DNS-over-HTTPS](https://developers.google.com/speed/public-dns/docs/dns-over-https).
It effectively encrypts all your DNS traffic.

The ultimate goal for the project is to provide a secure, caching DNS proxy that communicates with
recursive DNS resolvers over encrypted channels only. For now, it resolves DNS queries over
HTTPS/1.1, in a few independent threads. The plans for future include HTTP/2.0 and QUIC support,
better caching, and other resolvers (e.g. [OpenResolve](https://www.openresolve.com/) by OpenDNS).

## Quick start

Download a pre-built binary for your platform from [the latest
release](https://github.com/pforemski/dingo/releases/latest) (or build your own binaries).

Run dingo as root on port 53. For example, on Linux:
```
$ sudo ./dingo-linux-amd64 -port=53
```

Update your DNS configuration. On Linux, edit your `/etc/resolv.conf` as root (remember to
make backup first), e.g.:
```
$ sudo sh -c "echo nameserver 127.0.0.1 > /etc/resolv.conf"
```

## Tuning dingo

You will probably want to change the default Google DNS-over-HTTPS server IP address, using the
`-gdns:server` option. First, resolve `dns.google.com` to IP address, which should give you the
server closest to you:
```
$ host dns.google.com
dns.google.com has address 216.58.209.174
dns.google.com has IPv6 address 2a00:1450:401b:800::200e
```

Next, pass it to dingo. If you prefer IPv6, enclose the address in brackets, e.g.:
```
$ sudo ./dingo-linux-amd64 -port=53 -gdns:server=[2a00:1450:401b:800::200e]
```

To see all options, run `dingo -h`:
```
Usage of dingo:
  -bind string
    	IP address to bind to (default "0.0.0.0")
  -dbg int
    	debugging level (default 2)
  -gdns:edns string
    	Google DNS: EDNS client subnet (set 0.0.0.0/0 to disable)
  -gdns:nopad
    	Google DNS: disable random padding
  -gdns:server string
    	Google DNS: web server address (default "216.58.209.174")
  -gdns:sni string
    	Google DNS: SNI string to send (should match server certificate) (default "www.google.com")
  -gdns:workers int
    	Google DNS: number of independent workers (default 10)
  -port int
    	listen on port number (default 32000)
```

Note that by default dingo binds to all interfaces, which makes it open to the
world (unless you run a firewall). Consider binding it to `127.0.0.1` instead.

Finally, you will need to make dingo start in background each time you boot your machine. In Linux,
you might want to use the [GNU Screen](https://en.wikipedia.org/wiki/GNU_Screen), which can start
processes in background. For example, you might want to add the following line to your
`/etc/rc.local`:
```
screen -dmS dingo /path/to/bin/dingo -port=53 -bind=127.0.0.1 -gdns:server=[2a00:1450:401b:800::200e]
```

## Author

Pawel Foremski, [pjf@foremski.pl](mailto:pjf@foremski.pl)

Find me on: [LinkedIn](https://www.linkedin.com/in/pforemski),
[Twitter](https://twitter.com/pforemski)
