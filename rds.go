package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/gophercloud/utils/client"
	gophercloud "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v1/subnets"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v1/vpcs"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/security/groups"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v3/instances"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/http"
	"os"
)

const (
	AppVersion = "0.0.3"
	RdsYaml    = "rds.yaml"
)

type conf struct {
	Name             string          `yaml:"name"`
	Datastore        *Datastore      `yaml:"datastore"`
	Ha               *Ha             `yaml:"ha"`
	Port             string          `yaml:"port"`
	Password         string          `yaml:"password"`
	BackupStrategy   *BackupStrategy `yaml:"backupstrategy"`
	FlavorRef        string          `yaml:"flavorref"`
	Volume           *Volume         `yaml:"volume"`
	Region           string          `yaml:"region"`
	AvailabilityZone string          `yaml:"availabilityzone"`
	Vpc              string          `yaml:"vpc"`
	Subnet           string          `yaml:"subnet"`
	SecurityGroup    string          `yaml:"securitygroup"`
}

type Datastore struct {
	Type    string `json:"type" required:"true"`
	Version string `json:"version" required:"true"`
}

type Ha struct {
	Mode            string `json:"mode" required:"true"`
	ReplicationMode string `json:"replicationmode,omitempty"`
}

type BackupStrategy struct {
	StartTime string `json:"starttime" required:"true"`
	KeepDays  int    `json:"keepdays,omitempty"`
}

type Volume struct {
	Type string `json:"type" required:"true"`
	Size int    `json:"size" required:"true"`
}


func funcError(e string) {
        msg := errors.New(e)
	fmt.Println("ERROR:", msg)
	os.Exit(1)
	return
}

func secgroupGet(client *gophercloud.ServiceClient, opts *groups.ListOpts) (*groups.SecGroup, error) {

	pages,err := groups.List(client, *opts).AllPages()
	if err != nil {
		return nil, err
	}
	n, err := groups.ExtractGroups(pages)
	if len(n) == 0 {
		funcError("No SecurityGroups found")
	}

	return &n[0],nil
}

func subnetGet(client *gophercloud.ServiceClient, opts *subnets.ListOpts) (*subnets.Subnet, error) {

	n, err := subnets.List(client, *opts)
	if err != nil {
		return nil, err
	}
	if len(n) == 0 {
		funcError("No Subnet found")
	}

	return &n[0], nil
}

func vpcGet(client *gophercloud.ServiceClient, opts *vpcs.ListOpts) (*vpcs.Vpc, error) {

	n, err := vpcs.List(client, *opts)
	if err != nil {
		return nil, err
	}

	if len(n) == 0 {
		funcError("No VPC found")
	}

	return &n[0], nil
}

func rdsGet(client *gophercloud.ServiceClient, rdsId string) (*instances.RdsInstanceResponse, error) {

	listOpts := instances.ListRdsInstanceOpts{
		Id: rdsId,
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

func rdsCreate(netclient1 *gophercloud.ServiceClient, netclient2 *gophercloud.ServiceClient, client *gophercloud.ServiceClient, opts *instances.CreateRdsOpts) {

	var c conf
	c.getConf()

	g, err := secgroupGet(netclient2, &groups.ListOpts{Name: c.SecurityGroup})
	if err != nil {
		panic(err)
	}

	s, err := subnetGet(netclient1, &subnets.ListOpts{Name: c.Subnet})
	if err != nil {
		panic(err)
	}

	v, err := vpcGet(netclient1, &vpcs.ListOpts{Name: c.Vpc})
	if err != nil {
		panic(err)
	}

	createOpts := instances.CreateRdsOpts{
		Name: c.Name,
		Datastore: &instances.Datastore{
			Type:    c.Datastore.Type,
			Version: c.Datastore.Version,
		},
		Ha: &instances.Ha{
			Mode:            c.Ha.Mode,
			ReplicationMode: c.Ha.ReplicationMode,
		},
		Port:     c.Port,
		Password: c.Password,
		BackupStrategy: &instances.BackupStrategy{
			StartTime: c.BackupStrategy.StartTime,
			KeepDays:  c.BackupStrategy.KeepDays,
		},
		FlavorRef: c.FlavorRef,
		Volume: &instances.Volume{
			Type: c.Volume.Type,
			Size: c.Volume.Size,
		},
		Region:           c.Region,
		AvailabilityZone: c.AvailabilityZone,
		VpcId:            v.ID,
		SubnetId:         s.ID,
		SecurityGroupId:  g.ID,
	}

	createResult := instances.Create(client, createOpts)
	r, err := createResult.Extract()
	if err != nil {
		panic(err)
	}
	jobResponse, err := createResult.ExtractJobResponse()
	if err != nil {
		panic(err)
	}

	if err := instances.WaitForJobCompleted(client, int(1800), jobResponse.JobID); err != nil {
		panic(err)
	}

	rdsInstance, err := rdsGet(client, r.Instance.Id)

	fmt.Println(rdsInstance.PrivateIps[0])
	if err != nil {
		panic(err)
	}

	return
}

func (c *conf) getConf() *conf {

	yfile, err := ioutil.ReadFile(RdsYaml)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(yfile, c)
	if err != nil {
		panic(err)
	}

	return c
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

	network1, err := openstack.NewNetworkV1(provider, gophercloud.EndpointOpts{})
	network2, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{})
	rds, err := openstack.NewRDSV3(provider, gophercloud.EndpointOpts{})
	if err != nil {
		panic(err)
	}

	rdsCreate(network1, network2, rds, &instances.CreateRdsOpts{})
	if err != nil {
		panic(err)
	}
}
