# otc-rdsrestore-client

make a Point-In-Time-Recovery (PITR) of a RDS instance in OTC tenant based


## usage

provide cloud credentials based on env vars:

```bash
export OS_REGION_NAME=eu-de
export OS_AUTH_URL=https://iam.eu-de.otc.t-systems.com:443/v3
export OS_USERNAME=username
export OS_USER_DOMAIN_NAME=OTC-EU-DE-00000000000000000001
export OS_PROJECT_NAME=eu-de
export OS_PASSWORD=password
```

provide RDS name and timestamp were you want to restore (in UTC)

```bash
export RDS_NAME=mydb
export RDS_RESTORE_TIME=2022-02-08T22:00:00+00:00
```

start program

```bash
$ ./rdsrestore
```

That's it!

verify result

```bash
$ openstack rds instance show mydb
```

refer [OTC API DOC](https://docs.otc.t-systems.com/api/rds/rds_01_0002.html)


## Credits

Frank Kloeker f.kloeker@telekom.de

Life is for sharing. If you have an issue with the code or want to improve it, feel free to open an issue or an pull request.
