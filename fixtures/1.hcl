job "mysql" {
  region      = "eu"
  namespace   = "test"
  id          = "123"
  type        = "batch"
  priority    = 50
  all_at_once = true
  datacenters = ["a", "b"]

  constraint {
    attribute = "left"
    value     = "right"
    operator  = "op"
  }

  affinity {
    attribute = "left"
    value     = "right"
    operator  = "op"
    weight    = 10
  }

  group "mysqld" {
    task "server" {
      service {
        tags = ["leader", "mysql"]
        port = "db"

        check {
          type     = "tcp"
          port     = "db"
          interval = "10s"
          timeout  = "2s"
        }

        check {
          name     = "check_table"
          type     = "script"
          command  = "/usr/local/bin/check_mysql_table_status"
          args     = ["--verbose"]
          interval = "1m0s"
          timeout  = "5s"

          check_restart {
            limit = 3
            grace = "1m30s"
          }
        }
      }
    }

    restart {
      interval = "10m0s"
      attempts = 3
      delay    = "10s"
      mode     = "fail"
    }
  }

  meta {
    hi = "there"
  }
}
