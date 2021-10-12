job "countdash" {
  datacenters = ["dc1"]

  group "api" {
    task "web" {
      driver = "docker"

      config {
        image = "hashicorpnomad/counter-api:v3"
      }
    }

    network {
      mode = "bridge"
    }

    service {
      name = "count-api"
      port = "9001"

      check {
        name     = "api-health"
        type     = "http"
        path     = "/health"
        expose   = true
        interval = "10s"
        timeout  = "3s"
      }

      connect {
        sidecar_service {
        }
      }
    }
  }

  group "dashboard" {
    task "dashboard" {
      driver = "docker"

      config {
        image = "hashicorpnomad/counter-dashboard:v3"
      }

      env {
        COUNTING_SERVICE_URL = "http://${NOMAD_UPSTREAM_ADDR_count_api}"
      }
    }

    network {
      mode = "bridge"

      port "http" {
        static = 9002
        to     = 9002
      }
    }

    service {
      name = "count-dashboard"
      port = "9002"

      connect {
        sidecar_service {
          proxy {
            upstreams {
              destination_name = "count-api"
              local_bind_port  = 8080
            }
          }
        }
      }
    }
  }
}
