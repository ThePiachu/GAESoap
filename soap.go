// Original code by Hooklift - https://github.com/hooklift/gowsdl
// Modification for Google App Engine by ThePiachu

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package GAESoap

import (
	"appengine"
	"appengine/urlfetch"
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ThePiachu/Go/Log"
)


var timeout = time.Duration(30 * time.Second)

type SoapEnvelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	//Header SoapHeader `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header,omitempty"`
	Body SoapBody `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
}

type SoapHeader struct {
	Header interface{}
}

type SoapBody struct {
	Fault   *SoapFault `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault,omitempty"`
	Content string     `xml:",innerxml"`
}

type SoapFault struct {
	Faultcode   string `xml:"faultcode,omitempty"`
	Faultstring string `xml:"faultstring,omitempty"`
	Faultactor  string `xml:"faultactor,omitempty"`
	Detail      string `xml:"detail,omitempty"`
}

type SoapClient struct {
	url string
	tls bool
}

func (f *SoapFault) Error() string {
	return f.Faultstring
}

func NewSoapClient(url string, tls bool) *SoapClient {
	return &SoapClient{
		url: url,
		tls: tls,
	}
}

func (s *SoapClient) Call(c appengine.Context, soapAction string, request, response interface{}) error {
	envelope := SoapEnvelope{
	//Header:        SoapHeader{},
	}

	if request != nil {
		reqXml, err := xml.Marshal(request)
		if err != nil {
			Log.Errorf(c, "Call - %v", err)
			return err
		}

		envelope.Body.Content = string(reqXml)
	}
	buffer := &bytes.Buffer{}

	encoder := xml.NewEncoder(buffer)
	//encoder.Indent("  ", "    ")

	err := encoder.Encode(envelope)
	if err == nil {
		err = encoder.Flush()
	}
	if err != nil {
		Log.Errorf(c, "Call - %v", err)
		return err
	}

	//Log.Debugf(c, "buffer - %v", buffer)

	req, err := http.NewRequest("POST", s.url, buffer)
	req.Header.Add("Content-Type", "text/xml; charset=\"utf-8\"")
	if soapAction != "" {
		req.Header.Add("SOAPAction", soapAction)
	}
	req.Header.Set("User-Agent", "gowsdl/0.1")

	tr := urlfetch.Transport{
		Deadline: timeout,
		AllowInvalidServerCertificate: s.tls,
		Context: c,
	}

	client := &http.Client{Transport: &tr}
	res, err := client.Do(req)
	if err != nil {
		Log.Errorf(c, "Call - %v", err)
		return err
	}
	defer res.Body.Close()
	//Log.Debugf(c, "res - %v", res)

	rawbody, err := ioutil.ReadAll(res.Body)
	if len(rawbody) == 0 {
		Log.Warningf(c, "empty response")
		return nil
	}

	respEnvelope := &SoapEnvelope{}

	err = xml.Unmarshal(rawbody, respEnvelope)
	if err != nil {
		Log.Errorf(c, "Call - %v", err)
		Log.Debugf(c, "rawbody - %x", rawbody)
		Log.Debugf(c, "respEnvelope - %x", respEnvelope)
		return err
	}

	body := respEnvelope.Body.Content
	fault := respEnvelope.Body.Fault
	if body == "" {
		Log.Warningf(c, "empty response body", "envelope", respEnvelope, "body", body)
		return nil
	}

	Log.Debugf(c, "response", "envelope", respEnvelope, "body", body)
	if fault != nil {
		Log.Errorf(c, "Call - %v", fault)
		return fault
	}

	err = xml.Unmarshal([]byte(body), response)
	if err != nil {
		Log.Errorf(c, "Call - %v", err)
		return err
	}

	return nil
}
