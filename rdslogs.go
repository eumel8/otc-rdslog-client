package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gophercloud/utils/client"
	gophercloud "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v3/instances"
	"github.com/opentelekomcloud/gophertelekomcloud/pagination"
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

	slowLogOpts := instances.DbSlowLogOpts{StartDate: start_date, EndDate: end_date}
	// allPages, err := instances.ListSlowLog(client, slowLogOpts, rds.Id).AllPages()
	allPages, err := ListMySlowLog(client, slowLogOpts, rds.Id).AllPages()
	if err != nil {
		err := fmt.Errorf("error getting rds pages: %v", err)
		return err
	}
	// slowLogs, err := instances.ExtractSlowLog(allPages)
	slowLogs, err := ExtractSlowLog(allPages)
	if err != nil {
		err := fmt.Errorf("error getting rds slowlog: %v", err)
		return err
	}

	b, err := json.MarshalIndent(slowLogs.Slowloglist, "", "  ")
	if err != nil {
		err := fmt.Errorf("error marshal slowlog: %v", err)
		return err
	}

	fmt.Println(string(b))
	return nil
}

type ErrorLogPage struct {
	pagination.SinglePageBase
}

type SlowLogPage struct {
	pagination.SinglePageBase
}

type DbSlowLogBuilder interface {
	ToDbSlowLogListQuery() (string, error)
}

type SlowLogResp struct {
	Slowloglist []Slowloglist `json:"slow_log_list"`
	TotalRecord int           `json:"total_record"`
}

type Slowloglist struct {
	Count        string `json:"count"`
	Time         string `json:"time"`
	Locktime     string `json:"lock_time"`
	Rowssent     string `json:"rows_sent"`
	Rowsexamined string `json:"rows_examined"`
	Database     string `json:"database"`
	Users        string `json:"users"`
	QuerySample  string `json:"query_sample"`
	Type         string `json:"type"`
}

func ExtractSlowLog(r pagination.Page) (SlowLogResp, error) {
	var s SlowLogResp
	err := (r.(SlowLogPage)).ExtractInto(&s)
	return s, err
}

func listslowlogURL(c *gophercloud.ServiceClient, instanceID string) string {
	return c.ServiceURL("instances", instanceID, "slowlog")
}

func ListMySlowLog(client *gophercloud.ServiceClient, opts DbSlowLogBuilder, instanceID string) pagination.Pager {
	url := listslowlogURL(client, instanceID)
	if opts != nil {
		query, err := opts.ToDbSlowLogListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	pageRdsList := pagination.NewPager(client, url, func(r pagination.PageResult) pagination.Page {
		return SlowLogPage{pagination.SinglePageBase(r)}
	})

	rdsheader := map[string]string{"Content-Type": "application/json"}
	pageRdsList.Headers = rdsheader
	return pageRdsList
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
