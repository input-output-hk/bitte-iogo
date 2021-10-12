job "docs" {
  group "example" {
    task "server" {
      dispatch_payload {
        file = "config.json"
      }
    }
  }
}
