/**
 * dingo: a DNS caching proxy written in Go
 * This file implements an OpenDNS www.openresolve.com client
 *
 * Copyright (C) 2016 Pawel Foremski <pjf@foremski.pl>
 * Licensed under GNU GPL v3
 */

package main

import "fmt"
import "encoding/json"
import "time"
import "flag"
import "github.com/miekg/dns"

type OdnsQS struct {
	Qclass string
	Qtype  string
	Qname  string
}

type OdnsReply struct {
	ReturnCode string
	ID     int
	AA     bool
	AD     bool
	RA     bool
	RD     bool
	TC     bool
	QuestionSection OdnsQS
	AnswerSection     []interface{}
	AdditionalSection []interface{}
	AuthoritySection  []interface{}
}

/***********************************************************/

type Odns struct {
	workers *int
	server *string
	sni *string
	host *string
}

func (R *Odns) Init() {
	R.workers = flag.Int("odns:workers", 0,
		"OpenDNS: number of independent workers")
	R.server = flag.String("odns:server", "67.215.70.81",
		"OpenDNS: web server address")
	R.sni = flag.String("odns:sni", "www.openresolve.com",
		"OpenDNS: TLS SNI string to send (unencrypted, must validate as server cert)")
	R.host = flag.String("odns:host", "api.openresolve.com",
		"OpenDNS: HTTP 'Host' header (real FQDN, encrypted in TLS)")
}

func (R *Odns) Start() {
	if *R.workers <= 0 { return }

	dbg(1, "starting %d OpenDNS client(s) querying server %s",
		*R.workers, *R.server)
	for i := 0; i < *R.workers; i++ { go R.worker(*R.server) }
}

func (R *Odns) worker(server string) {
	var https = NewHttps(*R.sni)
	for q := range qchan { *q.rchan <- *R.resolve(https, server, q.Name, q.Type) }
}

func (R *Odns) resolve(https *Https, server string, qname string, qtype int) *Reply {
	r := Reply{ Status: -1 }

	/* prepare */
	uri := fmt.Sprintf("/%s/%s", dns.Type(qtype).String(), qname)

	/* query */
	buf, err := https.Get(server, *R.host, uri)
	if err != nil { return &r }
	r.Now = time.Now()

	/* parse */
	var f OdnsReply
	json.Unmarshal(buf, &f)
	dbg(1, "TODO: %+v", f)

	return &r
}

/* register module */
var _ = register("odns", new(Odns))
