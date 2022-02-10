package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/gophercloud/utils/client"
	gophercloud "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v3/backups"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v3/instances"
	"net/http"
	"os"
	"time"
)

const (
	AppVersion = "0.0.1"
)

func RdsError(e string) {
	msg := errors.New(e)
	fmt.Println("ERROR:", msg)
	os.Exit(1)
	return
}

func RdsGetName(client *gophercloud.ServiceClient, rdsName string) (*instances.RdsInstanceResponse, error) {

	listOpts := instances.ListRdsInstanceOpts{
		Name: rdsName,
	}
	allPages, err := instances.List(client, listOpts).AllPages()
	if err != nil {
		return nil, err
	}

	rdsList, err := instances.ExtractRdsInstances(allPages)
	if err != nil {
		return nil, err
	}
	if len(rdsList.Instances) == 0 {
		return nil, nil
	}
	return &rdsList.Instances[0], nil
}

func RdsRestore(client *gophercloud.ServiceClient, opts *backups.RestorePITROpts) {

	rawRestoretime := os.Getenv("RDS_RESTORE_TIME")
	if rawRestoretime == "" {
		RdsError("Missing variable RDS_RESTORE_TIME (e.g. 2020-04-04T22:08:41+00:00)")
	}

	rdsRestoredate, err := time.Parse(time.RFC3339, rawRestoretime)
	if err != nil {
		RdsError("Can't parse time format")
	}
	rdsRestoretime := rdsRestoredate.UnixMilli()

	rdsName := os.Getenv("RDS_NAME")

	if rdsName == "" {
		RdsError("Missing variable RDS_NAME (e.g. mydb)")
	}

	rds, err := RdsGetName(client, rdsName)
	restoreOpts := backups.RestorePITROpts{
		Source: backups.Source{
			InstanceID:  rds.Id,
			RestoreTime: rdsRestoretime,
			Type:        "timestamp",
		},
		Target: backups.Target{
			InstanceID: rds.Id,
		},
	}

	restoreResult := backups.RestorePITR(client, restoreOpts)
	restoredRds, err := restoreResult.Extract()
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

	fmt.Println("done", restoredRds.Instance.Id)

	return
}

func main() {

	version := flag.Bool("version", false, "app version")
	help := flag.Bool("help", false, "print out the help")

	flag.Parse()

	if *help {
		fmt.Println("Provide ENV variable to connect OTC: OS_PROJECT_NAME, OS_REGION_NAME, OS_AUTH_URL, OS_IDENTITY_API_VERSION, OS_USER_DOMAIN_NAME, OS_USERNAME, OS_PASSWORD, RDS_NAME, RDS_RESTORE_TIME")
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

	RdsRestore(rds, &backups.RestorePITROpts{})
	if err != nil {
		panic(err)
	}
}
