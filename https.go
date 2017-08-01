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

func NewHttps(sni string, forceh1 bool) *Https {
	H := Https{}

	/* TLS setup */
	tlscfg := &tls.Config{
		ServerName: sni,
		//Enforce highest known available Cipher
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305},
		MinVersion: tls.VersionTLS12,
	}

	/* HTTP transport */
	var tr http.RoundTripper
	switch {
	case forceh1 || *opt_h1:
		h1 := new(http.Transport)
		h1.TLSClientConfig = tlscfg
		tr = h1

	case *opt_quic:
		quic := new(h2quic.RoundTripper)
//		quic.TLSClientConfig = tlscfg // FIXME
		tr = quic

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
