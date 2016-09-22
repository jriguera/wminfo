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
	"time"

	"github.com/go-playground/log"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/units"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

// ShowVM represents a class to show VM properties
type ShowVM struct {
	*ListVMs
	vms []mo.VirtualMachine
	pc  *property.Collector
}

// NewShowVM is the constructor
func NewShowVM(u *url.URL, insecure bool, dc string, ctx context.Context) *ShowVM {
	showvm := ShowVM{}
	showvm.ListVMs = NewListVMs(u, insecure, dc, ctx)
	log.Debug("ListNets constructor")
	return &showvm
}

func (showvm *ShowVM) console(vm *mo.VirtualMachine) string {
	port := "7331"
	consoleURL := "http://%s:%s/console/?vmId=%s&vmName=%s&host=%s&sessionTicket=%s&thumbprint=%s"
	sessionTicket := showvm.clonesession()
	thumbprint := showvm.fingerprint()
	host := showvm.host()
	vcenter := showvm.url.Host
	vmID := vm.Reference().Value
	return fmt.Sprintf(consoleURL, vcenter, port, vmID, vm.Name, host, sessionTicket, thumbprint)
}

func (showvm *ShowVM) collectReferences(vm *mo.VirtualMachine) ([]mo.HostSystem, []mo.Network, []mo.DistributedVirtualPortgroup, []mo.Datastore) {
	var host []mo.HostSystem
	var network []mo.Network
	var dvp []mo.DistributedVirtualPortgroup
	var datastore []mo.Datastore

	// Table to drive inflating refs to their mo.* counterparts (dest)
	// and save() the Name to entities w/o using reflection here.
	// Note that we cannot use a []mo.ManagedEntity here, since mo.Network has its own 'Name' field,
	// the mo.Network.ManagedEntity.Name field will not be set.
	entities := make(map[types.ManagedObjectReference]string)
	vrefs := map[string]*struct {
		dest interface{}
		refs []types.ManagedObjectReference
		save func()
	}{
		"HostSystem": {
			&host, nil, func() {
				for _, e := range host {
					entities[e.Reference()] = e.Name
				}
			},
		},
		"Network": {
			&network, nil, func() {
				for _, e := range network {
					entities[e.Reference()] = e.Name
				}
			},
		},
		"DistributedVirtualPortgroup": {
			&dvp, nil, func() {
				for _, e := range dvp {
					entities[e.Reference()] = e.Name
				}
			},
		},
		"Datastore": {
			&datastore, nil, func() {
				for _, e := range datastore {
					entities[e.Reference()] = e.Name
				}
			},
		},
	}
	// Add MOR to vrefs[kind].refs avoiding any duplicates.
	addRef := func(refs ...types.ManagedObjectReference) {
		for _, ref := range refs {
			vref := vrefs[ref.Type]
			for _, r := range vref.refs {
				if r == ref {
					return
				}
			}
			vref.refs = append(vref.refs, ref)
		}
	}
	if ref := vm.Summary.Runtime.Host; ref != nil {
		addRef(*ref)
	}
	addRef(vm.Datastore...)
	addRef(vm.Network...)
	// Process the references
	for _, vref := range vrefs {
		if vref.refs != nil {
			if err := showvm.pc.Retrieve(showvm.ctx, vref.refs, []string{"name"}, vref.dest); err != nil {
				log.Panicf("Error retrieving resources references: %s", err)
			}
			vref.save()
		}
	}
	return host, network, dvp, datastore
}

// Print dumps a table with the results
func (showvm *ShowVM) Print(p ...string) {
	var vms []mo.VirtualMachine
	var fvms []mo.VirtualMachine
	var pvms *[]mo.VirtualMachine

	if len(p) == 0 {
		p = []string{"name", "summary", "guest", "config", "datastore", "network"}
	}
	if pc, err := property.DefaultCollector(showvm.base.client.Client).Create(showvm.base.ctx); err == nil {
		log.Debug("Printing information ...")
		showvm.pc = pc
		if err := pc.Retrieve(showvm.base.ctx, showvm.refs, p, &vms); err != nil {
			log.Panicf("Error retrieving resources information from references: %s", err)
		}
		// Filter the search here!
		// TODO improve this!
		if len(showvm.search) > 0 {
			for _, vm := range vms {
				for _, s := range showvm.search {
					if vm.Reference().Value == s ||
						strings.ToLower(vm.Name) == s ||
						(vm.Summary.Guest != nil &&
							(strings.ToLower(vm.Summary.Guest.HostName) == s ||
								strings.ToLower(vm.Summary.Guest.IpAddress) == s)) {
						fvms = append(fvms, vm)
						break
					}
				}
			}
			pvms = &fvms
		} else {
			pvms = &vms
		}
		tw := tabwriter.NewWriter(os.Stdout, 2, 0, 2, ' ', 0)
		fmt.Fprintf(tw, "VirtualMachine(s): %d\n", len(*pvms))
		fmt.Fprintf(tw, "---------------------\n")
		for _, vm := range *pvms {
			host, network, dvp, datastore := showvm.collectReferences(&vm)
			console := showvm.console(&vm)
			//fmt.Fprintf(tw, "VM:\t%s\n", vm)
			fmt.Fprintf(tw, "VM config\n")
			fmt.Fprintf(tw, "\tName:\t%s\n", vm.Name)
			fmt.Fprintf(tw, "\tId:\t%s\n", vm.Reference().Value)
			fmt.Fprintf(tw, "\tPath:\t%s\n", vm.Summary.Config.VmPathName)
			fmt.Fprintf(tw, "\tUUID:\t%s\n", vm.Summary.Config.Uuid)
			fmt.Fprintf(tw, "\tGuest: \t%s\n", vm.Summary.Config.GuestFullName)
			fmt.Fprintf(tw, "\tMemory:\t%d MB\n", vm.Summary.Config.MemorySizeMB)
			fmt.Fprintf(tw, "\tMemoryReservation:\t%d MB\n", vm.Summary.Config.MemoryReservation)
			fmt.Fprintf(tw, "\tCPU:\t%d vCPU(s)\n", vm.Summary.Config.NumCpu)
			fmt.Fprintf(tw, "\tCpuReservation:\t%d\n", vm.Summary.Config.CpuReservation)
			fmt.Fprintf(tw, "\tGuestId:\t%s\n", vm.Summary.Config.GuestId)
			fmt.Fprintf(tw, "\tInstanceUuid:\t%s\n", vm.Summary.Config.InstanceUuid)
			fmt.Fprintf(tw, "\tEthernetCards:\t%d\n", vm.Summary.Config.NumEthernetCards)
			fmt.Fprintf(tw, "\tVirtualDisks:\t%d\n", vm.Summary.Config.NumVirtualDisks)
			fmt.Fprintf(tw, "\tTemplate:\t%t\n", vm.Summary.Config.Template)
			if vm.Summary.Config.ManagedBy != nil {
				fmt.Fprintf(tw, "\tManagedBy:\t%s\n", vm.Summary.Config.ManagedBy.ExtensionKey)
			}
			fmt.Fprintf(tw, "\n")
			fmt.Fprintf(tw, "Guest\n")
			fmt.Fprintf(tw, "\tHostName: \t%s\n", vm.Summary.Guest.HostName)
			fmt.Fprintf(tw, "\tIpAddress: \t%s\n", vm.Summary.Guest.IpAddress)
			fmt.Fprintf(tw, "\tGuestId: \t%s\n", vm.Summary.Guest.GuestId)
			fmt.Fprintf(tw, "\tGuestFullName: \t%s\n", vm.Summary.Guest.GuestFullName)
			fmt.Fprintf(tw, "\tToolsRunningStatus: \t%s\n", vm.Summary.Guest.ToolsRunningStatus)
			fmt.Fprintf(tw, "\tToolsVersionStatus: \t%s\n", vm.Summary.Guest.ToolsVersionStatus)
			fmt.Fprintf(tw, "\n")
			fmt.Fprintf(tw, "Runtime env\n")
			if len(host) > 0 {
				fmt.Fprintf(tw, "\tHost:\t%s\n", host[0].Name)
				fmt.Fprintf(tw, "\tHostId:\t%s\n", vm.Summary.Runtime.Host.Value)
			}
			if vm.Summary.Runtime.BootTime != nil {
				fmt.Fprintf(tw, "\tBootTime:\t%s\n", vm.Summary.Runtime.BootTime)
			}
			fmt.Fprintf(tw, "\tPowerState: \t%s\n", vm.Summary.Runtime.PowerState)
			if vm.Summary.Runtime.PowerState != "poweredOn" {
				fmt.Fprintf(tw, "\tPaused:\t%t\n", vm.Summary.Runtime.Paused)
				fmt.Fprintf(tw, "\tCleanPowerOff:\t%t\n", vm.Summary.Runtime.CleanPowerOff)
				fmt.Fprintf(tw, "\tSuspendTime:\t%s\n", vm.Summary.Runtime.SuspendTime)
			}
			fmt.Fprintf(tw, "\tMemoryOverhead:\t%d MB\n", vm.Summary.Runtime.MemoryOverhead)
			fmt.Fprintf(tw, "\tMaxMemoryUsage:\t%d MB\n", vm.Summary.Runtime.MaxMemoryUsage)
			fmt.Fprintf(tw, "\tMaxCpuUsage:\t%d\n", vm.Summary.Runtime.MaxCpuUsage)
			if len(network) > 0 {
				fmt.Fprintf(tw, "\tNetwork(s):\n")
				for _, i := range network {
					fmt.Fprintf(tw, "\t\t%s: %s\n", i.Reference().Value, i.Name)
				}
			}
			if len(dvp) > 0 {
				fmt.Fprintf(tw, "\tVirtual Switch(s):\n")
				for _, i := range dvp {
					fmt.Fprintf(tw, "\t\t%s: %s\n", i.Reference().Value, i.Name)
				}
			}
			fmt.Fprintf(tw, "\n")
			fmt.Fprintf(tw, "Storage\n")
			fmt.Fprintf(tw, "\tUncommitted:\t%s\n", units.ByteSize(vm.Summary.Storage.Uncommitted))
			fmt.Fprintf(tw, "\tCommitted:\t%s\n", units.ByteSize(vm.Summary.Storage.Committed))
			fmt.Fprintf(tw, "\tUnshared:\t%s\n", units.ByteSize(vm.Summary.Storage.Unshared))
			fmt.Fprintf(tw, "\tDatastores:\n")
			for _, i := range datastore {
				fmt.Fprintf(tw, "\t\t%s: %s\n", i.Reference().Value, i.Name)
			}
			fmt.Fprintf(tw, "\n")
			fmt.Fprintf(tw, "QuickStats\n")
			fmt.Fprintf(tw, "\tOverallCpuDemand:\t%d\n", vm.Summary.QuickStats.OverallCpuDemand)
			fmt.Fprintf(tw, "\tOverallCpuUsage:\t%d\n", vm.Summary.QuickStats.OverallCpuUsage)
			fmt.Fprintf(tw, "\tBalloonedMemory:\t%d MB\n", vm.Summary.QuickStats.BalloonedMemory)
			fmt.Fprintf(tw, "\tCompressedMemory:\t%d MB\n", vm.Summary.QuickStats.CompressedMemory)
			fmt.Fprintf(tw, "\tConsumedOverheadMemory:\t%dMB\n", vm.Summary.QuickStats.ConsumedOverheadMemory)
			fmt.Fprintf(tw, "\tGuestMemoryUsage:\t%d MB\n", vm.Summary.QuickStats.GuestMemoryUsage)
			fmt.Fprintf(tw, "\tHostMemoryUsage:\t%d MB\n", vm.Summary.QuickStats.HostMemoryUsage)
			fmt.Fprintf(tw, "\tSwappedMemory:\t%d MB\n", vm.Summary.QuickStats.SwappedMemory)
			fmt.Fprintf(tw, "\tSharedMemory:\t%d MB\n", vm.Summary.QuickStats.SharedMemory)
			fmt.Fprintf(tw, "\tPrivateMemory:\t%d MB\n", vm.Summary.QuickStats.PrivateMemory)
			fmt.Fprintf(tw, "\tUptimeSeconds:\t%d s\n", vm.Summary.QuickStats.UptimeSeconds)
			fmt.Fprintf(tw, "\n")
			fmt.Fprintf(tw, "Annotations\n")
			for _, field := range strings.Fields(vm.Summary.Config.Annotation) {
				f := strings.SplitN(field, ":", 2)
				fmt.Fprintf(tw, "\t%s:\t%s\n", f[0], f[1])
			}
			//fmt.Fprintf(tw, "\n")
			//fmt.Fprintf(tw, "ExtraConfig\n")
			//for _, v := range vm.Config.ExtraConfig {
			//	fmt.Fprintf(tw, "\t%s:\t%s\n", v.GetOptionValue().Key, v.GetOptionValue().Value)
			//}
			fmt.Fprintf(tw, "\n")
			fmt.Fprintf(tw, "Console\n")
			fmt.Fprintf(tw, "You have 60 seconds to open the URL, or the session will be terminated.\n")
			fmt.Fprintf(tw, "\t%s\n", console)
		}
		tw.Flush()
	}
	fmt.Println()
	fmt.Println("Waiting for 60 seconds, then exit")
	time.Sleep(time.Second * 60)
}
