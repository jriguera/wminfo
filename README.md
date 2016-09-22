# wminfo

Golang program to retrieve information about VMWare VCenter resources:

* Dumps information about a specific VM and provides a link to open a 
  console without opening a session directly in VCenter
* Shows Information about VCenter and the list of DataCenters
* List of DataStores and DataStore Cluster with their storage capacity
* List of Network resources available in the Datacenter
* List of VirtualMachines: Reference, Name, GuestId, PowerState and IP

The information includes Annotations, which are useful when you have VM managed
by OpenStack Nova Vmware hypervisor.


```
Usage of wminfo:
        wminfo [OPTIONS] { info | ds | net | vms | show <VM name|IP|Reference> }

Show information about VMware VCenter resources

OPTIONS:
  -dc string
        Datacenter [WMINFO_DC]
  -debug
        No verify the server's certificate chain [WMINFO_DEBUG]
  -insecure
        No verify the server's certificate chain [WMINFO_INSECURE]
  -url string
        VCenter URL [WMINFO_URL] (default "https://username:password@host/sdk")

Instead of providing these OPTIONS, you can use the following environment variales:
        WMINFO_URL, WMINFO_USERNAME, WMINFO_PASSWORD
        WMINFO_DEBUG, WMINFO_INSECURE
        WMINFO_DC
```

# ScreenShots

```
wminfo -url https://user:pass@vcenter/sdk -dc Dordrecht -insecure show 10.100.15.10
VirtualMachine(s): 1
---------------------
VM config
  Name:               87552544-2842-4043-ad0a-72a5aec8a090
  Id:                 vm-96922
  Path:               [DATASTORE] 87552544-2842-4043-ad0a-72a5aec8a090/87552544-2842-4043-ad0a-72a5aec8a090.vmx
  UUID:               42254a47-db0a-34f6-be21-d888ca3e0261
  Guest:              Other (32-bit)
  Memory:             4096 MB
  MemoryReservation:  0 MB
  CPU:                2 vCPU(s)
  CpuReservation:     0
  GuestId:            otherGuest
  InstanceUuid:       87552544-2842-4043-ad0a-72a5aec8a090
  EthernetCards:      1
  VirtualDisks:       1
  Template:           false
  ManagedBy:          org.openstack.compute

Guest
  HostName:             jose-dev-mysql-01
  IpAddress:            10.100.15.10
  GuestId:              ubuntu64Guest
  GuestFullName:        Ubuntu Linux (64-bit)
  ToolsRunningStatus:   guestToolsRunning
  ToolsVersionStatus:   guestToolsUnmanaged

Runtime env
  Host:            esxi-42.springer-sbm.com
  HostId:          host-77028
  PowerState:      poweredOn
  MemoryOverhead:  49192960 MB
  MaxMemoryUsage:  4096 MB
  MaxCpuUsage:     5398
  Virtual Switch(s):
    dvportgroup-81396: brq0c4bbc5b-56

Storage
  Uncommitted:  38.9GB
  Committed:    45.2GB
  Unshared:     5.2GB
  Datastores:
    datastore-80906: DATASTORE

QuickStats
  OverallCpuDemand:        0
  OverallCpuUsage:         0
  BalloonedMemory:         0 MB
  CompressedMemory:        0 MB
  ConsumedOverheadMemory:  28MB
  GuestMemoryUsage:        0 MB
  HostMemoryUsage:         1162 MB
  SwappedMemory:           0 MB
  SharedMemory:            60 MB
  PrivateMemory:           1116 MB
  UptimeSeconds:           1409285 s

Annotations
  name:         jose-dev-mysql-01
  userid:       3fc9eae7edfe4d18b98b15d1f603e82e
  username:     jriguera
  projectid:    973bf207a89946aca3a2e3d78094d7cd
  projectname:  pe
  flavor:       name:medium
  flavor:       memory_mb:4096
  flavor:       vcpus:2
  flavor:       ephemeral_gb:0
  flavor:       root_gb:40
  flavor:       swap:0
  imageid:      271d650d-7d59-44a8-bc83-c959b99279f5
  package:      12.0.0

Console
You have 60 seconds to open the URL, or the session will be terminated.
  http://venter:7331/console/?vmId=vm-96922&vmName=87552544-2842-4043-ad0a-72a5aec8a090&host=vcenter&sessionTicket=cst-VCT-521f4bf0-36ed-6739-d34s-de580323c--tp-33-46-5F-EF-37-21-27-CD-CD-A3-AA-18-34-99-6C-4A-6E-91-1F-AA&thumbprint=33:46:5F:EF:37:21:27:CD:CD:A3:AA:18:34:99:6C:4A:6E:91:1F:AA

Waiting for 60 seconds, then exit

```

# Compile

* Install [glide](https://glide.sh/) package management
* Type `glide install`
* Type `go build`




