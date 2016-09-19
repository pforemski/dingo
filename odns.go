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
import "time"
import "io/ioutil"
import "encoding/json"
import "crypto/tls"
import "flag"
import "github.com/miekg/dns"

/* command-line arguments */
var (
	odns_workers = flag.Int("odns:workers", 0,
		"OpenDNS: number of independent workers")
	odns_server  = flag.String("odns:server", "67.215.70.81",
		"OpenDNS: web server address")
	odns_sni     = flag.String("odns:sni", "www.openresolve.com",
		"OpenDNS: SNI string to send (should match server certificate)")
)

/**********************************************************************/

func odns_start() {
	for i := 0; i < *odns_workers; i++ { go odns_resolver(*odns_server) }
}

func odns_resolver(server string) {
	/* setup the HTTP client */
	var httpTr = http.DefaultTransport.(*http.Transport)
	var tlsCfg = &tls.Config{ ServerName: *odns_sni }
	httpTr.TLSClientConfig = tlsCfg;
	var httpClient = &http.Client{ Timeout: time.Second*10, Transport: httpTr }

	for q := range qchan {
		/* make the new response object */
		r := Reply{ Status: -1 }

		/* prepare request, send proper HTTP 'Host:' header */
		addr := fmt.Sprintf("https://%s/%s/%s", server, dns.Type(q.Type).String(), q.Name)
		dbg(7, "  query: %s", addr)
		hreq,_ := http.NewRequest("GET", addr, nil)
		hreq.Host = "api.openresolve.com"

		/* send the query */
		resp,err := httpClient.Do(hreq)
		if (err == nil) {
			dbg(2, "[%s/%d] %s %s", q.Name, q.Type, resp.Status, resp.Proto)

			/* read */
			buf,_ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			dbg(7, "  reply: %s", buf)

			/* parse JSON? */
			if (resp.StatusCode == 200) { json.Unmarshal(buf, &r) }
			r.Now = time.Now()
		} else { dbg(1, "[%s/%d] error: %s", q.Name, q.Type, err.Error()) }

		/* write the reply */
		*q.rchan <- r
	}
}
