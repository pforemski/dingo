/**
 * dingo: a DNS caching proxy written in Go
 * This file implements a Google DNS-over-HTTPS client
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

type Gdns struct {
	workers *int
	server *string
	auto *bool
	auto6 *bool
	sni *string
	host *string
	edns *string
	nopad *bool
}

/* command-line arguments */
func (r *Gdns) Init() {
	r.workers = flag.Int("gdns:workers", 10,
		"Google DNS: number of independent workers")
	r.server  = flag.String("gdns:server", "216.58.195.78",
		"Google DNS: server address")
	r.auto   = flag.Bool("gdns:auto", false,
		"Google DNS: try to lookup the closest IPv4 server")
	r.auto6   = flag.Bool("gdns:auto6", false,
		"Google DNS: try to lookup the closest IPv6 server")
	r.sni     = flag.String("gdns:sni", "www.google.com",
		"Google DNS: SNI string to send (should match server certificate)")
	r.host    = flag.String("gdns:host", "dns.google.com",
		"Google DNS: HTTP 'Host' header (real FQDN, encrypted in TLS)")
	r.edns    = flag.String("gdns:edns", "",
		"Google DNS: EDNS client subnet (set 0.0.0.0/0 to disable)")
	r.nopad   = flag.Bool("gdns:nopad", false,
		"Google DNS: disable random padding")
}

/**********************************************************************/

func (R *Gdns) Start() {
	if *R.workers <= 0 { return }

	if *R.auto {
		dbg(1, "resolving dns.google.com...")
		r4 := R.resolve(NewHttps(*R.sni, false), *R.server, "dns.google.com", 1)
		if r4.Status == 0 && len(r4.Answer) > 0 {
			R.server = &r4.Answer[0].Data
		}
	}
	if *R.auto6 {
		dbg(1, "resolving dns.google.com...")
		r6 := R.resolve(NewHttps(*R.sni, false), *R.server, "dns.google.com", 28)
		if r6.Status == 0 && len(r6.Answer) > 0 {
			R.server = &r6.Answer[0].Data
		}
	}

	dbg(1, "starting %d Google Public DNS client(s) querying server %s",
		*R.workers, *R.server)
	for i := 0; i < *R.workers; i++ { go R.worker(*R.server) }
}

func (R *Gdns) worker(server string) {
	var https = NewHttps(*R.sni, false)
	for q := range qchan {
		*q.rchan <- *R.resolve(https, server, q.Name, q.Type)
	}
}

func (R *Gdns) resolve(https *Https, server string, qname string, qtype int) *Reply {
	r := Reply{ Status: -1 }
	v := url.Values{}

	/* prepare */
	v.Set("name", qname)
	v.Set("type", fmt.Sprintf("%d", qtype))
	if len(*R.edns) > 0 {
		v.Set("edns_client_subnet", *R.edns)
	}
	if !*R.nopad {
		v.Set("random_padding", strings.Repeat(string(65+rand.Intn(26)), rand.Intn(500)))
	}

	/* query */
	buf, err := https.Get(server, *R.host, "/resolve?" + v.Encode())
	if err != nil { return &r }

	/* parse */
	r.Now = time.Now()
	json.Unmarshal(buf, &r)

	return &r
}

/* register module */
var _ = register("gdns", new(Gdns))
