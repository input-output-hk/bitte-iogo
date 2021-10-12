job "ingress-demo" {
  datacenters = ["dc1"]

  group "ingress-group" {
    network {
      mode = "bridge"

      port "inbound" {
        static = 8080
        to     = 8080
      }
    }

    service {
      name = "my-ingress-service"
      port = "8080"

      connect {
        gateway {
          proxy {
            envoy_gateway_bind_addresses "address" {
              address = "0.0.0.0"
              port    = 12
            }
            envoy_gateway_no_default_bind = true
          }

          ingress {
            listener {
              port     = 8080
              protocol = "tcp"

              service {
                name = "uuid-api"
              }
            }
          }
        }
      }
    }
  }

  group "generator" {
    task "generate" {
      driver = "docker"

      config {
        image        = "hashicorpnomad/uuid-api:v5"
        network_mode = "host"
      }

      env {
        BIND = "0.0.0.0"
        PORT = "${NOMAD_PORT_api}"
      }
    }

    network {
      mode = "host"

      port "api" {
      }
    }

    service {
      name = "uuid-api"
      port = "api"

      connect {
        native = true
      }
    }
  }
}
