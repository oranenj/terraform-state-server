terraform {
  backend "http" {
    address = "http://localhost:8080/my/state.tf"
    lock_address = "http://localhost:8080/my/state.tf"
    unlock_address = "http://localhost:8080/my/state.tf"
  }
}

resource "null_resource" "foobar" {
	provisioner "local-exec" {
		command = "sleep 3"
	}
}
