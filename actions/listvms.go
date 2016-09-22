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
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

// ListVMs represents a class to list information about all vms and templates
// found in the datacenter
type ListVMs struct {
	*base
	refs   []types.ManagedObjectReference
	search []string
}

// NewListVMs is the constructor
func NewListVMs(u *url.URL, insecure bool, dc string, ctx context.Context) *ListVMs {
	listvms := ListVMs{}
	listvms.base = newBase(u, insecure, dc, ctx)
	log.Debug("ListVMs constructor")
	return &listvms
}

// Search gets vm references from vcenter.
// It accepts parameters to filter the search.
// It will return the number of references found.
func (listvms *ListVMs) Search(s ...string) int {
	var viewManager mo.ViewManager

	if len(s) == 0 {
		s = []string{"*"}
	}
	log.Debugf("Gathering VM references with filter: %s", strings.Join(s, ", "))
	for _, m := range s {
		listvms.search = append(listvms.search, m)
	}
	finder := find.NewFinder(listvms.client.Client, true)
	dc, err := finder.DatacenterOrDefault(listvms.ctx, listvms.dc)
	if err != nil {
		log.Panicf("Error getting datacenter: %s", err)
	}
	// http://www.geeklee.co.uk/object-properties-containerview-pyvmomi
	// Create the view manager
	if err := listvms.client.RetrieveOne(listvms.ctx, *listvms.client.ServiceContent.ViewManager, nil, &viewManager); err != nil {
		log.Panicf("Error creating viewManager: %s", err)
	}
	// Create the CreateContentView request
	req := types.CreateContainerView{
		This:      viewManager.Reference(),
		Container: dc.Reference(),
		Type:      []string{"VirtualMachine"},
		Recursive: true,
	}
	counter := 0
	if res, err := methods.CreateContainerView(listvms.ctx, listvms.client.RoundTripper, &req); err == nil {
		log.Debug("Getting list of vm references ...")
		var containerView mo.ContainerView
		if err := listvms.client.RetrieveOne(listvms.ctx, res.Returnval, nil, &containerView); err == nil {
			// Assign each MORS type to a specific array
			for _, mor := range containerView.View {
				if mor.Type == "VirtualMachine" {
					listvms.refs = append(listvms.refs, mor)
					counter++
				}
			}
		} else {
			log.Panicf("Error retrieving references: %s", err)
		}
	} else {
		log.Panicf("Error creating container view: %s", err)
	}
	return counter
}

// Print dumps a table with the results
func (listvms *ListVMs) Print(p ...string) {
	var vms []mo.VirtualMachine

	if len(p) == 0 {
		p = []string{"name", "summary"}
	}
	if pc, err := property.DefaultCollector(listvms.client.Client).Create(listvms.ctx); err == nil {
		log.Debug("Printing information ...")
		tw := tabwriter.NewWriter(os.Stdout, 1, 0, 1, ' ', 0)
		fmt.Fprintf(tw, "\n")
		fmt.Fprintf(tw, "Reference\tName\tHostName\tGuest\tPowerState\tIpAddress\n")
		fmt.Fprintf(tw, "---------\t----\t--------\t-----\t----------\t---------\n")
		if err := pc.Retrieve(listvms.ctx, listvms.refs, p, &vms); err != nil {
			log.Errorf("Error retrieving information from references: %s", err)
		} else {
			for _, vm := range vms {
				fmt.Fprintf(tw, "%s\t", vm.Reference().Value)
				fmt.Fprintf(tw, "%s\t", strings.SplitN(vm.Name, " ", 2)[0])
				if contains("summary", p) {
					fmt.Fprintf(tw, "%s\t", vm.Summary.Guest.HostName)
					fmt.Fprintf(tw, "%s\t", vm.Summary.Guest.GuestId)
					fmt.Fprintf(tw, "%s\t", vm.Summary.Runtime.PowerState)
					fmt.Fprintf(tw, "%s\t", vm.Summary.Guest.IpAddress)
				}
				fmt.Fprintf(tw, "\n")
			}
		}
		fmt.Fprintf(tw, "\n")
		tw.Flush()
	} else {
		log.Errorf("Error creating collector: %s", err)
	}
}
