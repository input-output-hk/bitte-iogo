job "test" {
  group "test" {
    task "test" {
      template {
        source      = "foo"
        destination = "bar"
        data        = "Something with \"quotes\" is fun"
      }
    }
  }
}
