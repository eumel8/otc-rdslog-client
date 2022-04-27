package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gophercloud/utils/client"
	gophercloud "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
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

func RdsErrorlog(client *gophercloud.ServiceClient) error {
	rdsName := os.Getenv("RDS_NAME")
	if rdsName == "" {
		err := fmt.Errorf("Missing variable RDS_NAME (e.f.mydb)")
		return err
	}

	sd := time.Now().AddDate(0, -1, 0)
	ed := time.Now()
	start_date := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d+0000",
		sd.Year(), sd.Month(), sd.Day(),
		sd.Hour(), sd.Minute(), sd.Second())
	end_date := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d+0000",
		ed.Year(), ed.Month(), ed.Day(),
		ed.Hour(), ed.Minute(), ed.Second())

	rds, err := RdsGetName(client, rdsName)

	errorLogOpts := instances.DbErrorlogOpts{StartDate: start_date, EndDate: end_date}
	allPages, err := instances.ListErrorLog(client, errorLogOpts, rds.Id).AllPages()
	if err != nil {
		err := fmt.Errorf("error getting rds pages: %v", err)
		return err
	}
	errorLogs, err := instances.ExtractErrorLog(allPages)
	if err != nil {
		err := fmt.Errorf("error getting rds errorlog: %v", err)
		return err
	}
	// fmt.Println("print the logs ", rds.Name)

	b, err := json.MarshalIndent(errorLogs.ErrorLogList, "", "  ")
	if err != nil {
		err := fmt.Errorf("error marshal errorlog: %v", err)
		return err
	}

	fmt.Println(string(b))
	return nil
}

func RdsSlowlog(client *gophercloud.ServiceClient) error {
	rdsName := os.Getenv("RDS_NAME")
	if rdsName == "" {
		err := fmt.Errorf("Missing variable RDS_NAME (e.f.mydb)")
		return err
	}

	sd := time.Now().AddDate(0, -1, 0)
	ed := time.Now()
	start_date := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d+0000",
		sd.Year(), sd.Month(), sd.Day(),
		sd.Hour(), sd.Minute(), sd.Second())
	end_date := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d+0000",
		ed.Year(), ed.Month(), ed.Day(),
		ed.Hour(), ed.Minute(), ed.Second())

	rds, err := RdsGetName(client, rdsName)

	slowLogOpts := instances.DbSlowlogOpts{StartDate: start_date, EndDate: end_date}
	allPages, err := instances.ListSlowLog(client, slowLogOpts, rds.Id).AllPages()
	if err != nil {
		err := fmt.Errorf("error getting rds pages: %v", err)
		return err
	}
	slowLogs, err := instances.ExtractSlowLog(allPages)
	if err != nil {
		err := fmt.Errorf("error getting rds slowlog: %v", err)
		return err
	}

	b, err := json.MarshalIndent(slowLogs.SlowLogList, "", "  ")
	if err != nil {
		err := fmt.Errorf("error marshal slowlog: %v", err)
		return err
	}

	fmt.Println(string(b))
	return nil
}

func main() {

	version := flag.Bool("version", false, "app version")
	help := flag.Bool("help", false, "print out the help")
	errorlog := flag.Bool("errorlog", false, "fetch errorlog")
	slowlog := flag.Bool("slowlog", false, "fetch slowlog")

	flag.Parse()

	if *help {
		fmt.Println("Provide ENV variable to connect OTC: OS_PROJECT_NAME, OS_REGION_NAME, OS_AUTH_URL, OS_IDENTITY_API_VERSION, OS_USER_DOMAIN_NAME, OS_USERNAME, OS_PASSWORD, RDS_NAME")
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

	if os.Getenv("OS_DEBUG") == "1" {
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

	if *errorlog {
		err := RdsErrorlog(rds)
		if err != nil {
			panic(err)
		}
	}

	if *slowlog {
		err := RdsSlowlog(rds)
		if err != nil {
			panic(err)
		}
	}
}
