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

type OdnsReply struct {
	ReturnCode        string
	ID                int
	AA                bool
	AD                bool
	RA                bool
	RD                bool
	TC                bool
	QuestionSection   map[string]interface{}
	AnswerSection     []map[string]interface{}
	AdditionalSection []map[string]interface{}
	AuthoritySection  []map[string]interface{}
}

/***********************************************************/

type Odns struct {
	workers *int
	server  *string
	sni     *string
	host    *string

	string2rcode map[string]int
	string2rtype map[string]uint16
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

	R.string2rcode = make(map[string]int)
	for rcode, str := range dns.RcodeToString {
		R.string2rcode[str] = rcode
	}

	R.string2rtype = make(map[string]uint16)
	for rtype, str := range dns.TypeToString {
		R.string2rtype[str] = rtype
	}
}

/* start OpenDNS workers */
func (R *Odns) Start() {
	if *R.workers <= 0 {
		return
	}

	dbg(1, "starting %d OpenDNS client(s) querying server %s",
		*R.workers, *R.server)
	for i := 0; i < *R.workers; i++ {
		go R.worker(*R.server)
	}
}

/* handler of new requests */
func (R *Odns) worker(server string) {
	var https = NewHttps(*R.sni, true)
	for q := range qchan {
		*q.rchan <- *R.resolve(https, server, q.Name, q.Type)
	}
}

/* resolve single request */
func (R *Odns) resolve(https *Https, server string, qname string, qtype int) *Reply {
	r := Reply{Status: -1}

	/* prepare */
	uri := fmt.Sprintf("/%s/%s", dns.Type(qtype).String(), qname)

	/* query */
	buf, err := https.Get(server, *R.host, uri)
	if err != nil {
		return &r
	}
	r.Now = time.Now()

	/* parse */
	var f OdnsReply
	json.Unmarshal(buf, &f)

	/* rewrite */
	r.Status = R.string2rcode[f.ReturnCode]
	r.TC = f.TC
	r.RD = f.RD
	r.RA = f.RA
	r.AD = f.AD
	r.CD = false

	for _, v := range f.AnswerSection {
		rr := R.odns2grr(v)
		if rr != nil {
			r.Answer = append(r.Answer, *rr)
		}
	}

	for _, v := range f.AdditionalSection {
		rr := R.odns2grr(v)
		if rr != nil {
			r.Additional = append(r.Additional, *rr)
		}
	}

	for _, v := range f.AuthoritySection {
		rr := R.odns2grr(v)
		if rr != nil {
			r.Authority = append(r.Authority, *rr)
		}
	}

	return &r
}

func (R *Odns) odns2grr(v map[string]interface{}) *GRR {
	/* catch panics */
	defer func() {
		if r := recover(); r != nil {
			dbg(1, "panic in odns2grr()")
		}
	}()

	/* get basic data */
	rname := v["Name"].(string)
	rtypes := v["Type"].(string)
	rttl := uint32(v["TTL"].(float64))

	/* parse type & data */
	var rdata string
	var rtype uint16
	switch rtypes {
	case "A":
		rtype = dns.TypeA
		rdata = v["Address"].(string)
	case "AAAA":
		rtype = dns.TypeAAAA
		rdata = v["Address"].(string)
	case "CNAME":
		rtype = dns.TypeCNAME
		rdata = v["Target"].(string)
	case "MX":
		rtype = dns.TypeMX
		mx := v["MailExchanger"].(string)
		pref := v["Preference"].(float64)
		rdata = fmt.Sprintf("%d %s", int(pref), mx)
	case "NS":
		rtype = dns.TypeNS
		rdata = v["Target"].(string)
	case "NAPTR":
		rtype = dns.TypeNAPTR
		flg := v["Flags"].(string)
		ord := v["Order"].(float64)
		svc := v["Service"].(string)
		prf := v["Preference"].(float64)
		reg := v["Regexp"].(string)
		rep := v["Replacement"].(string)
		rdata = fmt.Sprintf("%d %d \"%s\" \"%s\" \"%s\" %s",
			int(ord), int(prf), flg, svc, reg, rep)
	case "PTR":
		rtype = dns.TypePTR
		rdata = v["Target"].(string)
	case "SOA":
		rtype = dns.TypeSOA
		msn := v["MasterServerName"].(string)
		mn := v["MaintainerName"].(string)
		ser := v["Serial"].(float64)
		ref := v["Refresh"].(float64)
		ret := v["Retry"].(float64)
		exp := v["Expire"].(float64)
		nttl := v["NegativeTtl"].(float64)
		rdata = fmt.Sprintf("%s %s %d %d %d %d %d",
			msn, mn, int(ser), int(ref), int(ret), int(exp), int(nttl))
	case "TXT":
		rtype = dns.TypeTXT
		rdata = v["TxtData"].(string)
	default:
		dbg(1, "odns2grr(): %s unsupported", rtypes)
		return nil
	}

	return &GRR{rname, rtype, rttl, rdata}
}

/* register module */
var _ = register("odns", new(Odns))
