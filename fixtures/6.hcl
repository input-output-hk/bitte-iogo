job "docs" {
  group "example" {
    task "server" {
      resources {
        device "nvidia/gpu" {
          count = 2

          constraint {
            attribute = "${device.attr.memory}"
            value     = "2 GiB"
            operator  = ">="
          }

          affinity {
            attribute = "${device.attr.memory}"
            value     = "4 GiB"
            operator  = ">="
            weight    = 75
          }
        }
      }
    }
  }
}
