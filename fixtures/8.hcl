job "docs" {
  group "example" {
    ephemeral_disk {
      sticky  = true
      migrate = true
      size    = 500
    }
  }
}
