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
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/go-playground/log"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

// VCInfo represents a class to gather basic info from Vmware Vcenter
type VCInfo struct {
	*base
	about *types.AboutInfo
	refs  []types.ManagedObjectReference
}

// NewVCInfo constructor
func NewVCInfo(u *url.URL, insecure bool, dc string, ctx context.Context) *VCInfo {
	vc := VCInfo{}
	vc.base = newBase(u, insecure, dc, ctx)
	log.Debug("VCInfo constructor")
	return &vc
}

// Search gets information from VCenter: available clusters.
// The parameter accepted represents the base path to perform the search.
// It will return the number of references found.
func (vc *VCInfo) Search(s ...string) int {

	if len(s) == 0 {
		s = []string{"*"}
	}
	log.Debugf("Gathering VCenter information with pattern: %s", strings.Join(s, ", "))
	vc.about = &vc.client.ServiceContent.About
	counter := 0
	finder := find.NewFinder(vc.client.Client, true)
	if datacenters, err := finder.DatacenterList(vc.ctx, s[0]); err == nil {
		log.Debugf("Getting list of datacenters")
		for _, dc := range datacenters {
			vc.refs = append(vc.refs, dc.Reference())
			counter++
		}
	} else {
		log.Panicf("Error getting datacenters references: %s", err)
	}
	return counter
}

// Print dumps basic VCenter info and the list of datacenters
func (vc *VCInfo) Print(p ...string) {
	var dcs []mo.Datacenter

	if len(p) == 0 {
		p = []string{"name"}
	}
	log.Debug("Printing information ...")
	tw := tabwriter.NewWriter(os.Stdout, 2, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "\n")
	fmt.Fprintf(tw, "About\n")
	fmt.Fprintf(tw, "-----\n")
	fmt.Fprintf(tw, "Name:\t%s\n", vc.about.Name)
	fmt.Fprintf(tw, "Vendor:\t%s\n", vc.about.Vendor)
	fmt.Fprintf(tw, "Version:\t%s\n", vc.about.Version)
	fmt.Fprintf(tw, "Build:\t%s\n", vc.about.Build)
	fmt.Fprintf(tw, "OS type:\t%s\n", vc.about.OsType)
	fmt.Fprintf(tw, "API type:\t%s\n", vc.about.ApiType)
	fmt.Fprintf(tw, "API version:\t%s\n", vc.about.ApiVersion)
	fmt.Fprintf(tw, "Product ID:\t%s\n", vc.about.ProductLineId)
	fmt.Fprintf(tw, "UUID:\t%s\n", vc.about.InstanceUuid)
	if pc, err := property.DefaultCollector(vc.client.Client).Create(vc.ctx); err == nil {
		fmt.Fprintf(tw, "\n")
		fmt.Fprintf(tw, "Datacenters\n")
		fmt.Fprintf(tw, "-----------\n")
		if err := pc.Retrieve(vc.ctx, vc.refs, p, &dcs); err != nil {
			log.Errorf("Error retrieving datacenter properties: %s", err)
		} else {
			for _, dc := range dcs {
				fmt.Fprintf(tw, "%s\t", dc.Reference())
				fmt.Fprintf(tw, "%s\t", dc.Name)
				fmt.Fprintf(tw, "\n")
			}
		}
	} else {
		log.Errorf("Error creating collector: %s", err)
	}
	fmt.Fprintf(tw, "\n")
	tw.Flush()
}
