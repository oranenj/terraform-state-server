# Terraform state server over HTTP

## Usage

```
# Run on localhost port 8080
terraform-state-server sqlite://states.db 8080
```
See tests/state.tf for an example of how to configure it

The state server will listen to requests on 127.0.0.1 on the specified port, and this is intentionally not configurable. 
Use a HTTP proxy with authentication to secure it.

Note: this has been only very lightly tested, but seems to work fine.
