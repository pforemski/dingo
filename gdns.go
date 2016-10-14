/**
 * dingo: a DNS caching proxy written in Go
 * This file implements a Google DNS-over-HTTPS client
 *
 * Copyright (C) 2016 Pawel Foremski <pjf@foremski.pl>
 * Licensed under GNU GPL v3
 */

package main

import "fmt"
import "net/http"
import "net/url"
import "time"
import "io/ioutil"
import "encoding/json"
import "crypto/tls"
import "math/rand"
import "strings"
import "flag"
//import "github.com/devsisters/goquic"

/* command-line arguments */
var (
	gdns_workers = flag.Int("gdns:workers", 10,
		"Google DNS: number of independent workers")
	gdns_server  = flag.String("gdns:server", "216.58.209.174",
		"Google DNS: web server address")
	gdns_sni     = flag.String("gdns:sni", "www.google.com",
		"Google DNS: SNI string to send (should match server certificate)")
	gdns_edns    = flag.String("gdns:edns", "",
		"Google DNS: EDNS client subnet (set 0.0.0.0/0 to disable)")
	gdns_nopad   = flag.Bool("gdns:nopad", false,
		"Google DNS: disable random padding")
)

/**********************************************************************/

func gdns_start() {
	for i := 0; i < *gdns_workers; i++ { go gdns_resolver(*gdns_server) }
}

func gdns_resolver(server string) {
	/* setup the HTTP client */
	//var httpTr = http.DefaultTransport.(*http.Transport)
	var httpTr = new(http.Transport)
//	var httpTr = goquic.NewRoundTripper(true)

	var tlsCfg = &tls.Config{ ServerName: *gdns_sni }
	httpTr.TLSClientConfig = tlsCfg;
//	req,_ := http.NewRequest("GET", "https://www.google.com/", nil)
//	httpTr.RoundTrip(req)

	var httpClient = &http.Client{ Timeout: time.Second*10, Transport: httpTr }

	for q := range qchan {
		/* make the new response object */
		r := Reply{ Status: -1 }

		/* prepare the query */
		v := url.Values{}
		v.Set("name", q.Name)
		v.Set("type", fmt.Sprintf("%d", q.Type))
		if len(*gdns_edns) > 0 {
			v.Set("edns_client_subnet", *gdns_edns)
		}
		if !*gdns_nopad {
			v.Set("random_padding", strings.Repeat(string(65+rand.Intn(26)), rand.Intn(500)))
		}

		/* prepare request, send proper HTTP 'Host:' header */
		addr     := fmt.Sprintf("https://%s/resolve?%s", server, v.Encode())
		hreq,_   := http.NewRequest("GET", addr, nil)
		hreq.Host = "dns.google.com"

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
