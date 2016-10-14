/**
 * dingo: a DNS caching proxy written in Go
 * This file implements an OpenDNS www.openresolve.com client
 *
 * Copyright (C) 2016 Pawel Foremski <pjf@foremski.pl>
 * Licensed under GNU GPL v3
 */

package main

import "fmt"
import "net/http"
import "crypto/tls"
import "io/ioutil"
import "encoding/json"
import "time"
import "flag"
import "github.com/miekg/dns"

type Odns struct {
	workers *int
	server *string
	sni *string
	host *string
}

func (r *Odns) Init() {
	r.workers = flag.Int("odns:workers", 0,
		"OpenDNS: number of independent workers")

	r.server = flag.String("odns:server", "67.215.70.81",
		"OpenDNS: web server address")

	r.sni = flag.String("odns:sni", "www.openresolve.com",
		"OpenDNS: TLS SNI string to send (unencrypted, must validate as server cert)")

	r.host = flag.String("odns:host", "api.openresolve.com",
		"OpenDNS: HTTP 'Host' header (real FQDN, encrypted in TLS)")
}

func (r *Odns) Start() {
	dbg(1, "starting %d OpenDNS clients", *r.workers)
	for i := 0; i < *r.workers; i++ {
		go r.worker(*r.server, *r.sni, *r.host)
	}
}

func (r *Odns) worker(ip string, sni string, host string) {
	/* setup the HTTP client */
	var httpTr = http.DefaultTransport.(*http.Transport)
	var tlsCfg = &tls.Config{ ServerName: sni }
	httpTr.TLSClientConfig = tlsCfg;
	var httpClient = &http.Client{ Timeout: time.Second*10, Transport: httpTr }

	for q := range qchan {
		/* make the new response object */
		r := Reply{ Status: -1 }

		/* prepare request, send proper HTTP 'Host:' header */
		addr     := fmt.Sprintf("https://%s/%s/%s", ip, dns.Type(q.Type).String(), q.Name)
		hreq,_   := http.NewRequest("GET", addr, nil)
		hreq.Host = host

		/* send the query */
		resp,err := httpClient.Do(hreq)
		if (err == nil) {
			dbg(2, "[%s/%d] %s %s", q.Name, q.Type, resp.Status, resp.Proto)

			/* read */
			buf,_ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			dbg(7, "  reply: %s", buf)

			/* parse JSON? */
			var f interface{}
			if (resp.StatusCode == 200) {
				json.Unmarshal(buf, &f)
				dbg(1, "TODO: %+v", f)
			}
			r.Now = time.Now()
		} else { dbg(1, "[%s/%d] error: %s", q.Name, q.Type, err.Error()) }

		/* write the reply */
		*q.rchan <- r
	}
}

/* register module */
var _ = mod_register("odns", new(Odns))
