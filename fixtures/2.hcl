job "docs" {
  group "example" {
    task "server" {
      artifact {
        source = "https://example.com/file.tar.gz"

        options {
          checksum = "md5:df6a4178aec9fbdc1d6d7e3634d1bc33"
          depth    = "1"
        }

        headers {
          User-Agent    = "nomad-[${NOMAD_JOB_ID}]-[${NOMAD_GROUP_NAME}]-[${NOMAD_TASK_NAME}]"
          X-Nomad-Alloc = "${NOMAD_ALLOC_ID}"
        }
        destination = "local/some-directory"
      }
    }
  }
}
