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
	"github.com/vmware/govmomi/units"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

// ListDSs is class to list DataStores
type ListDSs struct {
	*base
	refs        []types.ManagedObjectReference
	clusterRefs []types.ManagedObjectReference
}

// NewListDSs is the constructor
func NewListDSs(u *url.URL, insecure bool, dc string, ctx context.Context) *ListDSs {
	listdss := ListDSs{}
	listdss.base = newBase(u, insecure, dc, ctx)
	log.Debug("ListDSs constructor")
	return &listdss
}

// Search gets DataStore and DataStoreCluster references from VCenter.
// It accepts one parameter to filter the search.
// It will return the number of references found.
func (listdss *ListDSs) Search(s ...string) int {

	if len(s) == 0 {
		s = []string{"*"}
	}
	log.Debugf("Gathering Datastore references with filter: %s", strings.Join(s, ", "))
	finder := find.NewFinder(listdss.client.Client, true)
	if dc, err := finder.DatacenterOrDefault(listdss.ctx, listdss.dc); err != nil {
		log.Panicf("Error getting datacenter: %s", err)
	} else {
		finder.SetDatacenter(dc)
	}
	// Find DataStores in datacenter
	counter := 0
	if dss, err := finder.DatastoreList(listdss.ctx, s[0]); err != nil {
		log.Panicf("Error retrieving datastore list: %s", err)
	} else {
		log.Debug("Getting list of datastore references")
		// Convert DSs into list of references
		for _, ds := range dss {
			listdss.refs = append(listdss.refs, ds.Reference())
			counter++
		}
	}
	if dscs, err := finder.DatastoreClusterList(listdss.ctx, s[0]); err != nil {
		log.Panicf("Error retrieving datastore cluster list: %s", err)
	} else {
		log.Debug("Getting list of datastore cluster references")
		for _, ds := range dscs {
			listdss.clusterRefs = append(listdss.clusterRefs, ds.Reference())
			counter++
		}
	}
	return counter
}

// Print dumps a table with the results
func (listdss *ListDSs) Print(p ...string) {
	var dsts []mo.Datastore
	var dstsc []mo.StoragePod

	if len(p) == 0 {
		p = []string{"summary"}
	}
	if pc, err := property.DefaultCollector(listdss.client.Client).Create(listdss.ctx); err == nil {
		log.Debug("Printing information ...")
		tw := tabwriter.NewWriter(os.Stdout, 2, 0, 2, ' ', 0)
		fmt.Fprintf(tw, "\n")
		fmt.Fprintf(tw, "Reference\tName\tType\tCapacity\tFreeSpace\n")
		fmt.Fprintf(tw, "---------\t----\t----\t--------\t---------\n")
		if err := pc.Retrieve(listdss.ctx, listdss.refs, p, &dsts); err != nil {
			log.Errorf("Error retrieving datastore information from references: %s", err)
		} else {
			for _, dst := range dsts {
				fmt.Fprintf(tw, "%s\t", dst.Reference())
				fmt.Fprintf(tw, "%s\t", dst.Name)
				if contains("summary", p) {
					//fmt.Fprintf(tw, "%s\t", dst.Summary.Name)
					fmt.Fprintf(tw, "%s\t", dst.Summary.Type)
					fmt.Fprintf(tw, "%s\t", units.ByteSize(dst.Summary.Capacity))
					fmt.Fprintf(tw, "%s\t", units.ByteSize(dst.Summary.FreeSpace))
				}
				fmt.Fprintf(tw, "\n")
			}
		}
		if err := pc.Retrieve(listdss.ctx, listdss.clusterRefs, p, &dstsc); err != nil {
			log.Errorf("Error retrieving datastore cluster information from references: %s", err)
		} else {
			for _, dst := range dstsc {
				fmt.Fprintf(tw, "%s\t", dst.Reference())
				fmt.Fprintf(tw, "%s\t", dst.Name)
				if contains("summary", p) {
					//fmt.Fprintf(tw, "%s\t", dst.Summary.Name)
					fmt.Fprintf(tw, "%s\t", "-")
					fmt.Fprintf(tw, "%s\t", units.ByteSize(dst.Summary.Capacity))
					fmt.Fprintf(tw, "%s\t", units.ByteSize(dst.Summary.FreeSpace))
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
