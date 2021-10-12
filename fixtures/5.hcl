job "docs" {
  constraint {
    attribute = "${attr.kernel.name}"
    value     = "linux"
  }

  group "example" {
    constraint {
      value    = "true"
      operator = "distinct_hosts"
    }

    task "server" {
      constraint {
        attribute = "${meta.my_custom_value}"
        value     = "3"
        operator  = ">"
      }
    }
  }
}
