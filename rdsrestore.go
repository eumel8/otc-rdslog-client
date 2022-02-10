package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/gophercloud/utils/client"
	gophercloud "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v3/instances"
	"hiller.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v3/backups"
	"net/http"
	"os"
	"time"
)

const (
	AppVersion = "0.0.1"
)

func funcError(e string) {
	msg := errors.New(e)
	fmt.Println("ERROR:", msg)
	os.Exit(1)
	return
}

func rdsGetName(client *gophercloud.ServiceClient, rdsName string) (*instances.RdsInstanceResponse, error) {

	listOpts := instances.ListRdsInstanceOpts{
		Name: rdsName,
	}
	allPages, err := instances.List(client, listOpts).AllPages()
	if err != nil {
		return nil, err
	}

	n, err := instances.ExtractRdsInstances(allPages)
	if err != nil {
		return nil, err
	}
	if len(n.Instances) == 0 {
		return nil, nil
	}
	return &n.Instances[0], nil
}

func rdsRestore(client *gophercloud.ServiceClient, opts *backups.RestorePITROpts) {

	rawrestoretime := os.Getenv("RDS_RESTORE_TIME")
	if rawrestoretime == "" {
		funcError("Missing variable RDS_RESTORE_TIME (e.g. 2020-04-04T22:08:41+00:00)")
	}

	rdsrestoredate, err := time.Parse(time.RFC3339, rawrestoretime)
	if err != nil {
		funcError("Can't parse time format")
	}
	rdsrestoretime := rdsrestoredate.UnixMilli()

	rdsname := os.Getenv("RDS_RESTORE_DB")

	if rdsname == "" {
		funcError("Missing variable RDS_RESTORE_DB (e.g. mydb)")
	}

	rdsid,err := rdsGetName(client, rdsname)
	restoreOpts := backups.RestorePITROpts{
		Source: backups.Source{
			InstanceID:  rdsid.Id,
			RestoreTime: rdsrestoretime,
			Type:        "timestamp",
		},
		Target: backups.Target{
			InstanceID: rdsid.Id,
		},
	}

	restoreResult := backups.RestorePITR(client, restoreOpts)
	r, err := restoreResult.Extract()
	if err != nil {
		panic(err)
	}

	jobResponse, err := restoreResult.ExtractJobResponse()
	if err != nil {
		panic(err)
	}

	if err := instances.WaitForJobCompleted(client, int(1800), jobResponse.JobID); err != nil {
		panic(err)
	}

	fmt.Println("done",r.Instance.Id)

	return
}

func main() {

	version := flag.Bool("version", false, "app version")
	help := flag.Bool("help", false, "print out the help")

	flag.Parse()

	if *help {
		fmt.Println("Provide ENV variable to connect OTC: OS_PROJECT_NAME, OS_REGION_NAME, OS_AUTH_URL, OS_IDENTITY_API_VERSION, OS_USER_DOMAIN_NAME, OS_USERNAME, OS_PASSWORD")
		os.Exit(0)
	}

	if *version {
		fmt.Println("version", AppVersion)
		os.Exit(0)
	}

	if os.Getenv("OS_AUTH_URL") == "" {
		os.Setenv("OS_AUTH_URL", "https://iam.eu-de.otc.t-systems.com:443/v3")
	}

	if os.Getenv("OS_IDENTITY_API_VERSION") == "" {
		os.Setenv("OS_IDENTITY_API_VERSION", "3")
	}

	if os.Getenv("OS_REGION_NAME") == "" {
		os.Setenv("OS_REGION_NAME", "eu-de")
	}

	if os.Getenv("OS_PROJECT_NAME") == "" {
		os.Setenv("OS_PROJECT_NAME", "eu-de")
	}

	opts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		panic(err)
	}

	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		panic(err)
	}

	if os.Getenv("OS_DEBUG") != "" {
		provider.HTTPClient = http.Client{
			Transport: &client.RoundTripper{
				Rt:     &http.Transport{},
				Logger: &client.DefaultLogger{},
			},
		}
	}

	rds, err := openstack.NewRDSV3(provider, gophercloud.EndpointOpts{})
	if err != nil {
		panic(err)
	}

	rdsRestore(rds, &backups.RestorePITROpts{})
	if err != nil {
		panic(err)
	}
}
