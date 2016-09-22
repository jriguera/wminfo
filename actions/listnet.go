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

// ListNets represents a class to list network resources
type ListNets struct {
	*base
	refsNet  []types.ManagedObjectReference
	refsDVPG []types.ManagedObjectReference
	refsDVS  []types.ManagedObjectReference
}

// NewListNets is the constructor
func NewListNets(u *url.URL, insecure bool, dc string, ctx context.Context) *ListNets {
	listnets := ListNets{}
	listnets.base = newBase(u, insecure, dc, ctx)
	log.Debug("ListNets constructor")
	return &listnets
}

// Search gets network resources references from VCenter.
// It accepts one parameter to filter the search.
// It will return the number of references found.
func (listnets *ListNets) Search(s ...string) int {

	if len(s) == 0 {
		s = []string{"*"}
	}
	log.Debugf("Gathering Network resources references with filter: %s", strings.Join(s, ", "))
	finder := find.NewFinder(listnets.client.Client, true)
	if dc, err := finder.DatacenterOrDefault(listnets.ctx, listnets.dc); err != nil {
		log.Panicf("Error getting datacenter: %s", err)
	} else {
		finder.SetDatacenter(dc)
	}
	// Find DataStores in datacenter
	counter := 0
	if nets, err := finder.NetworkList(listnets.ctx, s[0]); err != nil {
		log.Panicf("Error retrieving network list: %s", err)
	} else {
		// Convert Nets into list of references
		log.Debug("Getting list of network references")
		for _, n := range nets {
			switch n.Reference().Type {
			case "Network":
				listnets.refsNet = append(listnets.refsNet, n.Reference())
				counter++
			case "VmwareDistributedVirtualSwitch":
				listnets.refsDVS = append(listnets.refsDVS, n.Reference())
				counter++
			case "DistributedVirtualPortgroup":
				listnets.refsDVPG = append(listnets.refsDVPG, n.Reference())
				counter++
			}
		}
	}
	return counter
}

// Print dumps a table with the results
func (listnets *ListNets) Print(p ...string) {
	var nets []mo.Network
	var dvs []mo.DistributedVirtualSwitch
	var dvpg []mo.DistributedVirtualPortgroup

	if len(p) == 0 {
		p = []string{"summary"}
	}
	if pc, err := property.DefaultCollector(listnets.client.Client).Create(listnets.ctx); err == nil {
		log.Debug("Printing information ...")
		tw := tabwriter.NewWriter(os.Stdout, 2, 0, 2, ' ', 0)
		fmt.Fprintf(tw, "\n")
		fmt.Fprintf(tw, "Reference\tName\tAccessible\n")
		fmt.Fprintf(tw, "---------\t----\t----------\n")
		if err := pc.Retrieve(listnets.ctx, listnets.refsNet, p, &nets); err != nil {
			log.Errorf("Error retrieving Network information from references: %s", err)
		} else {
			for _, net := range nets {
				s := net.Summary.GetNetworkSummary()
				fmt.Fprintf(tw, "%s\t", net.Reference())
				fmt.Fprintf(tw, "%s\t", s.Name)
				fmt.Fprintf(tw, "%t\t", s.Accessible)
				fmt.Fprintf(tw, "\n")
			}
		}
		if err := pc.Retrieve(listnets.ctx, listnets.refsDVPG, p, &dvpg); err != nil {
			log.Errorf("Error retrieving DVPG information from references: %s", err)
		} else {
			for _, net := range dvpg {
				s := net.Summary.GetNetworkSummary()
				fmt.Fprintf(tw, "%s\t", net.Reference())
				fmt.Fprintf(tw, "%s\t", s.Name)
				fmt.Fprintf(tw, "%t\t", s.Accessible)
				fmt.Fprintf(tw, "\n")
			}
		}
		if err := pc.Retrieve(listnets.ctx, listnets.refsDVS, p, &dvs); err != nil {
			log.Errorf("Error retrieving DVS information from references: %s", err)
		} else {
			fmt.Fprintf(tw, "\n")
			for _, net := range dvs {
				fmt.Fprintf(tw, "%s\t", net.Reference())
				fmt.Fprintf(tw, "%s\t", net.Name)
				fmt.Fprintf(tw, "-\t")
				if contains("summary", p) {
					fmt.Fprintf(tw, "\n")
					for _, pg := range net.Summary.PortgroupName {
						fmt.Fprintf(tw, "    %s\n", pg)
					}
				} else {
					fmt.Fprintf(tw, "\n")
				}
			}
		}
		fmt.Fprintf(tw, "\n")
		tw.Flush()
	}
}
