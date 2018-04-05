/**
 * dingo: a DNS caching proxy written in Go
 * This file implements a Cloudflare DNS-over-HTTPS client
 *
 * Copyright (C) 2016 Pawel Foremski <pjf@foremski.pl>
 * Licensed under GNU GPL v3
 */

package main

import "fmt"
import "net/url"
import "time"
import "encoding/json"
import "math/rand"
import "strings"
import "flag"

/* API Docs https://developers.cloudflare.com/1.1.1.1/dns-over-https/request-structure/ */

type Cfdns struct {
	workers *int
	server *string
	auto *bool
	sni *string
	host *string
	edns *string
	nopad *bool
}

/* command-line arguments */
func (r *Cfdns) Init() {
	r.workers = flag.Int("cfdns:workers", 10,
		"Cloudflare DNS: number of independent workers")
	r.server  = flag.String("cfdns:server", "1.1.1.1", /* or 1.0.0.1 */
		"Cloudflare DNS: server address")
	r.auto   = flag.Bool("cfdns:auto", false,
		"Cloudflare DNS: try to lookup the closest IPv4 server")
	r.sni     = flag.String("cfdns:sni", "dns.cloudflare.com",
		"Cloudflare DNS: SNI string to send (should match server certificate)")
	r.host    = flag.String("cfdns:host", "dns.cloudflare.com",
		"Cloudflare DNS: HTTP 'Host' header (real FQDN, encrypted in TLS)")
	r.edns    = flag.String("cfdns:edns", "",
		"Cloudflare DNS: EDNS client subnet (set 0.0.0.0/0 to disable)")
	r.nopad   = flag.Bool("cfdns:nopad", false,
		"Cloudflare DNS: disable random padding")
}

/**********************************************************************/

func (R *Cfdns) Start() {
	if *R.workers <= 0 { return }

	if *R.auto {
		dbg(1, "resolving dns.cloudflare.com...")
		r4 := R.resolve(NewHttps(*R.sni, false), *R.server, "dns.cloudflare.com", 1)
		if r4.Status == 0 && len(r4.Answer) > 0 {
			R.server = &r4.Answer[0].Data
		}
	}

	dbg(1, "starting %d Cloudflare Public DNS client(s) querying server %s",
		*R.workers, *R.server)
	for i := 0; i < *R.workers; i++ { go R.worker(*R.server) }
}

func (R *Cfdns) worker(server string) {
	var https = NewHttps(*R.sni, false)
	for q := range qchan {
		*q.rchan <- *R.resolve(https, server, q.Name, q.Type)
	}
}

func (R *Cfdns) resolve(https *Https, server string, qname string, qtype int) *Reply {
	r := Reply{ Status: -1 }
	v := url.Values{}

	/* prepare */
    v.Set("ct", "application/dns-json") /* cfdns special: must set content type here */
	v.Set("name", qname)
	v.Set("type", fmt.Sprintf("%d", qtype))
	if len(*R.edns) > 0 {
		v.Set("edns_client_subnet", *R.edns)
	}
	if !*R.nopad {
		v.Set("random_padding", strings.Repeat(string(65+rand.Intn(26)), rand.Intn(500)))
	}

	/* query */
	buf, err := https.Get(server, *R.host, "/dns-query?" + v.Encode())
	if err != nil { return &r }

	/* parse */
	r.Now = time.Now()
	json.Unmarshal(buf, &r)

	return &r
}

/* register module */
var _ = register("cfdns", new(Cfdns))
