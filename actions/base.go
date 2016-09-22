/*
Copyright (c) 2016 Jose Riguera Lopez. All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package actions

import (
	"crypto/sha1"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-playground/log"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

// Action Interface methods for all actions
type Action interface {
	Search(s ...string) int
	Print(p ...string)
}

func contains(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

type base struct {
	client *govmomi.Client
	url    *url.URL
	dc     string
	ctx    context.Context
}

// newBase is the constructor
func newBase(u *url.URL, insecure bool, dc string, ctx context.Context) *base {
	c, err := govmomi.NewClient(ctx, u, insecure)
	if err != nil {
		log.Panicf("Cannot connect with %s: %s", u, err)
	}
	b := base{c, u, dc, ctx}
	log.Infof("Connected to %s. Using datacenter %s", u, dc)
	return &b
}

func (b *base) clonesession() string {
	var token string

	gclient := b.client
	req := types.AcquireCloneTicket{
		This: gclient.SessionManager.Reference(),
	}
	if res, err := methods.AcquireCloneTicket(b.ctx, gclient.RoundTripper, &req); err == nil {
		token = res.Returnval
		log.Debugf("Cloning current session from %s: %s ", b.url, token)
	} else {
		log.Errorf("Error cloning session tiket from %s: %s", b.url, err)
	}
	return token
}

func (b *base) fingerprint() string {
	var fingerpring []string

	// Do a HEAD request to Vcenter to know the TLS sha1 fingerprint
	// otherwise it could be decoded from the ticket response, but this
	// is the right way to do it
	log.Debugf("Getting SSL fingerprint from %s", b.client.URL().String())
	if response, err := b.client.Head(b.client.URL().String()); err == nil {
		rawfingerprint := fmt.Sprintf("%x", sha1.Sum(response.TLS.PeerCertificates[0].Raw))
		log.Debugf("FingerPrint from %s: %s", b.client.URL().String(), rawfingerprint)
		for i := 0; i < len(rawfingerprint)-1; i = i + 2 {
			fingerpring = append(fingerpring, strings.ToUpper(rawfingerprint[i:i+2]))
		}
	} else {
		log.Panicf("Error getting SSL fingerprint from %s: %s", b.client.URL().String(), err)
	}
	return strings.Join(fingerpring, ":")
}

func (b *base) host() string {
	return b.client.Client.Client.URL().Host
}
