job "docs" {
  affinity {
    attribute = "${node.datacenter}"
    value     = "us-west1"
    weight    = 100
  }

  group "example" {
    affinity {
      attribute = "${meta.rack}"
      value     = "r1"
      weight    = 50
    }

    task "server" {
      affinity {
        attribute = "${meta.my_custom_value}"
        value     = "3"
        operator  = ">"
        weight    = 50
      }
    }
  }
}
