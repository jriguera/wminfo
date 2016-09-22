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

package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/go-playground/log"
	"github.com/go-playground/log/handlers/console"
	"github.com/jriguera/wminfo/actions"
	"golang.org/x/net/context"
)

const (
	envURL      = "WMINFO_URL"
	envUserName = "WMINFO_USERNAME"
	envPassword = "WMINFO_PASSWORD"
	envInsecure = "WMINFO_INSECURE"
	envDC       = "WMINFO_DC"
	envDebug    = "WMINFO_DEBUG"
)

// GetEnvString returns string from environment variable.
func GetEnvString(v string, def string) string {
	r := os.Getenv(v)
	if r == "" {
		return def
	}
	return r
}

// GetEnvBool returns boolean from environment variable.
func GetEnvBool(v string, def bool) bool {
	r := os.Getenv(v)
	if r == "" {
		return def
	}
	switch strings.ToLower(r[0:1]) {
	case "t", "y", "1":
		return true
	}
	return false
}

// EnvOverride overrides the auth parameters (user and pass) from the provided URL
// with the variables from env
func EnvOverride(u *url.URL) {
	envUsername := GetEnvString(envUserName, "")
	envPassword := GetEnvString(envPassword, "")

	// Override username if provided
	if envUsername != "" {
		if u.User != nil {
			password, ok := u.User.Password()
			if ok {
				u.User = url.UserPassword(envUsername, password)
			} else {
				u.User = url.User(envUsername)
			}
		}
	}
	// Override password if provided
	if envPassword != "" {
		if u.User != nil {
			u.User = url.UserPassword(u.User.Username(), envPassword)
		} else {
			u.User = url.UserPassword("", envPassword)
		}
	}
}

func main() {
	// https://blog.golang.org/defer-panic-and-recover
	// http://dahernan.github.io/2015/02/04/context-and-cancellation-of-goroutines/
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Flag (Args)
	urlDescription := fmt.Sprintf("VCenter URL [%s]", envURL)
	urlFlag := flag.String("url", GetEnvString(envURL, "https://username:password@host/sdk"), urlDescription)
	insecureDescription := fmt.Sprintf("No verify the server's certificate chain [%s]", envInsecure)
	insecureFlag := flag.Bool("insecure", GetEnvBool(envInsecure, false), insecureDescription)
	dcDescription := fmt.Sprintf("Datacenter [%s]", envDC)
	dcFlag := flag.String("dc", GetEnvString(envDC, ""), dcDescription)
	debugDescription := fmt.Sprintf("No verify the server's certificate chain [%s]", envDebug)
	debugFlag := flag.Bool("debug", GetEnvBool(envDebug, false), debugDescription)
	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Printf("\t%s [OPTIONS] { info | ds | net | vms | show <VM name|IP|Reference> }\n\n", os.Args[0])
		fmt.Printf("Show information about VMware VCenter resources\n\n")
		fmt.Println("OPTIONS:")
		flag.PrintDefaults()
		fmt.Printf("\nInstead of providing these OPTIONS, you can use the following environment variales:\n")
		fmt.Printf("\tWMINFO_URL, WMINFO_USERNAME, WMINFO_PASSWORD\n")
		fmt.Printf("\tWMINFO_DEBUG, WMINFO_INSECURE\n")
		fmt.Printf("\tWMINFO_DC\n\n")
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}
	// Logging
	cLog := console.New()
	d := []log.Level{log.ErrorLevel, log.PanicLevel, log.AlertLevel, log.FatalLevel}
	if *debugFlag {
		d = log.AllLevels
	}
	log.RegisterHandler(cLog, d...)
	// Parse URL from string
	u, err := url.Parse(*urlFlag)
	if err == nil {
		// Override username and/or password as required
		EnvOverride(u)
	} else {
		log.Panicf("Error parsing URL %s: %s", *urlFlag, err)
	}
	// Parse the command
	var a actions.Action
	switch flag.Arg(0) {
	case "info":
		a = actions.NewVCInfo(u, *insecureFlag, *dcFlag, ctx)
		a.Search("*")
	case "ds":
		a = actions.NewListDSs(u, *insecureFlag, *dcFlag, ctx)
		a.Search("*")
	case "net":
		a = actions.NewListNets(u, *insecureFlag, *dcFlag, ctx)
		a.Search("*")
	case "vms":
		a = actions.NewListVMs(u, *insecureFlag, *dcFlag, ctx)
		a.Search("*")
	case "show":
		if flag.Arg(1) != "" {
			a = actions.NewShowVM(u, *insecureFlag, *dcFlag, ctx)
			a.Search(flag.Arg(1))
		} else {
			flag.Usage()
			os.Exit(1)
		}
	default:
		flag.Usage()
		os.Exit(1)
	}
	a.Print()
	os.Exit(0)
}
