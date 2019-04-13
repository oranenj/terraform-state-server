terraform {
  backend "http" {
    address = "http://localhost:8080/"
    lock_address = "http://localhost:8080/"
    unlock_address = "http://localhost:8080/"
  }
}

resource "null_resource" "foobar" {
	provisioner "local-exec" {
		command = "sleep 3"
	}
}
