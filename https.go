/**
 * dingo: a DNS caching proxy written in Go
 * This file implements common code for HTTPS+JSON requests
 *
 * Copyright (C) 2016-2017 Pawel Foremski <pjf@foremski.pl>
 * Licensed under GNU GPL v3
 */

package main

import "time"
import "net/http"
import "io/ioutil"
import "crypto/tls"
import "errors"
import "golang.org/x/net/http2"
import "github.com/lucas-clemente/quic-go/h2quic"

type Https struct {
	client     http.Client
}

func NewHttps(sni string) *Https {
	H := Https{}

	/* TLS setup */
	tlscfg := new(tls.Config)
	tlscfg.ServerName = sni

	/* HTTP transport */
	var tr http.RoundTripper
	switch {
	case *opt_quic:
		quic := new(h2quic.QuicRoundTripper)
//		quic.TLSClientConfig = tlscfg // FIXME
		tr = quic
	case *opt_h1:
		h1 := new(http.Transport)
		h1.TLSClientConfig = tlscfg
		tr = h1
	default:
		h2 := new(http2.Transport)
		h2.TLSClientConfig = tlscfg
		tr = h2
	}

	/* HTTP client */
	H.client.Timeout = time.Second * 10
	H.client.Transport = tr

	return &H
}

func (R *Https) Get(ip string, host string, uri string) ([]byte, error) {
	url := "https://" + ip + uri
	hreq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		dbg(1, "http.NewRequest(): %s", err)
		return nil, err
	}
	hreq.Host = host // FIXME: doesn't have an effect for QUIC

	/* send the query */
	resp, err := R.client.Do(hreq)
	if err != nil {
		dbg(1, "http.Do(): %s", err)
		return nil, err
	}
	dbg(3, "http.Do(%s): %s %s", url, resp.Status, resp.Proto)

	/* read */
	buf, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		dbg(1, "ioutil.ReadAll(%s): %s", url, err)
		return nil, err
	}
	dbg(7, "  reply: %s", buf)

	/* HTTP 200 OK? */
	if resp.StatusCode != 200 {
		dbg(1, "resp.StatusCode != 200: %s", url)
		return nil, errors.New("response code != 200")
	}

	return buf, nil
}
