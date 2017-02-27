/**
 * dingo: a DNS caching proxy written in Go
 * This file implements common code for HTTPS+JSON requests
 *
 * Copyright (C) 2016-2017 Pawel Foremski <pjf@foremski.pl>
 * Licensed under GNU GPL v3
 */

package main

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lucas-clemente/quic-go/h2quic"
	"golang.org/x/net/http2"
	"golang.org/x/net/proxy"
)

type Https struct {
	client http.Client
}

func NewHttps(sni string, forceh1 bool) *Https {
	H := Https{}

	/* TLS setup */
	tlscfg := new(tls.Config)
	tlscfg.ServerName = sni
	tlscfg.InsecureSkipVerify = *opt_insecure

	/* HTTP transport */
	var tr http.RoundTripper
	forceh1 = forceh1 || len(*opt_proxy) > 0 // Force h1 to support proxy
	switch {
	case forceh1 || *opt_h1:
		h1 := &http.Transport{
			TLSClientConfig: tlscfg,
		}
		if proxyURL, err := url.Parse(*opt_proxy); err != nil {
			dbg(1, "proxyURL = url.Parse(): %s", err)
		} else {
			switch strings.ToUpper(proxyURL.Scheme) {
			case "HTTP":
				h1.Proxy = func(_ *http.Request) (*url.URL, error) {
					return proxyURL, nil
				}
			case "SOCKS5":
				fallthrough
			case "SOCKS":
				dialer, _ := proxy.SOCKS5("tcp", proxyURL.Host, nil, proxy.Direct)
				h1.Dial = dialer.Dial
			}
		}
		tr = h1

	case *opt_quic:
		quic := &h2quic.QuicRoundTripper{
		// TLSClientConfig: tlscfg, // FIXME
		}
		tr = quic

	default:
		h2 := &http2.Transport{
			TLSClientConfig: tlscfg,
		}
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
