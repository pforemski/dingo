# dingo
A DNS client (stub resolver) implemented in Go for the [Google
DNS-over-HTTPS](https://developers.google.com/speed/public-dns/docs/dns-over-https).
It effectively encrypts all your DNS traffic. It also supports
[OpenResolve](https://www.openresolve.com/) by OpenDNS.

The ultimate goal for the project is to provide a secure, caching DNS client that
communicates with recursive DNS resolvers over encrypted channels only. For now,
it resolves DNS queries over HTTP/2 in independent threads. The plans for
future include better caching and support for QUIC.

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
Usage of dingo-linux-amd64:
  -bind string
    	IP address to bind to (default "127.0.0.1")
  -dbg int
    	debugging level (default 2)
  -gdns:auto
    	Google DNS: try to lookup the closest IPv4 server
  -gdns:edns string
    	Google DNS: EDNS client subnet (set 0.0.0.0/0 to disable)
  -gdns:host string
    	Google DNS: HTTP 'Host' header (real FQDN, encrypted in TLS) (default "dns.google.com")
  -gdns:nopad
    	Google DNS: disable random padding
  -gdns:server string
    	Google DNS: server address (default "216.58.195.78")
  -gdns:sni string
    	Google DNS: SNI string to send (should match server certificate) (default "www.google.com")
  -gdns:workers int
    	Google DNS: number of independent workers (default 10)
  -h1
    	use HTTPS/1.1 transport
  -h1:proxy string
    	use Proxy of HTTP or SOCKS5, (Example "http://127.0.0.1:8080" or "socks(5)://127.0.0.1:1080")
  -insecure
    	disable SSL Certificate check
  -nocache
    	disable DNS Cache
  -odns:host string
    	OpenDNS: HTTP 'Host' header (real FQDN, encrypted in TLS) (default "api.openresolve.com")
  -odns:server string
    	OpenDNS: web server address (default "67.215.70.81")
  -odns:sni string
    	OpenDNS: TLS SNI string to send (unencrypted, must validate as server cert) (default "www.openresolve.com")
  -odns:workers int
    	OpenDNS: number of independent workers
  -port int
    	listen on port number (default 32000)

```

Finally, you will need to make dingo start in background each time you boot your machine. In Linux,
you might want to use the [GNU Screen](https://en.wikipedia.org/wiki/GNU_Screen), which can start
processes in background. For example, you might want to add the following line to your
`/etc/rc.local`:
```
screen -dmS dingo /path/to/bin/dingo -port=53 -gdns:server=[2a00:1450:401b:800::200e]
```

## Author

Pawel Foremski, [pjf@foremski.pl](mailto:pjf@foremski.pl)

Find me on: [LinkedIn](https://www.linkedin.com/in/pforemski),
[Twitter](https://twitter.com/pforemski)
