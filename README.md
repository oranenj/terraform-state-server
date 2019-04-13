# Terraform state server over HTTP

## Usage

```
sqlite3 states.db < tests/init.sql
terraform-state-server sqlite://states.db
```

The state server will listen to requests on 127.0.0.1:8080. Use a HTTP proxy with authentication to secure it.


Note: this has been only very lightly tested, but seems to work fine.
