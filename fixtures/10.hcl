job "docs" {
  group "example" {
    task "example" {
      volume_mount {
        volume      = "certs"
        destination = "/etc/ssl/certs"
      }
    }

    volume "certs" {
      type      = "host"
      source    = "ca-certificates"
      read_only = true
    }
  }
}
