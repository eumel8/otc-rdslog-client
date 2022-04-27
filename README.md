# otc-rdslog-client

print out RDS logs to STDOUT


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

provide RDS name 

```bash
export RDS_NAME=mydb
```

start program

fetch errorlogs (last 30 days)

```bash
$ ./rdsrestore -errorlogs
```

fetch slowlogs (last 30 days)


```bash
$ ./rdsrestore -slowlogs
```

That's it!


## Credits

Frank Kloeker f.kloeker@telekom.de

Life is for sharing. If you have an issue with the code or want to improve it, feel free to open an issue or an pull request.
