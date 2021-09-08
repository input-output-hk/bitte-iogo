job "db-sync-mainnet" {
  namespace   = "catalyst-sync"
  id          = "db-sync-mainnet"
  type        = "service"
  priority    = 50
  datacenters = ["eu-central-1", "us-east-2", "eu-west-1"]
  constraint {
    attribute = "$${attr.unique.platform.aws.instance-id}"
    value     = "i-0cdb0448a6d48f717"
    operator  = "="
  }

  group "db-sync" {
    count = 1

    task "db-sync" {
      driver = "exec"
      config {
        flake   = "github:input-output-hk/cardano-db-sync?rev=af6f4d31d137388aa59bae10c2fa79c219ce433d#cardano-db-sync-extended-mainnet"
        command = "/bin/cardano-db-sync-extended-entrypoint"
      }
      env = {
        CARDANO_NODE_SOCKET_PATH = "/alloc/node.socket"
        PATH                     = "/bin"
      }
      resources {
        cpu    = 3600
        memory = 12288
      }
      volume_mount {
        volume           = "persist"
        destination      = "/persist"
        read_only        = false
        propagation_mode = "private"
      }
      leader         = false
      shutdown_delay = "0s"
      kill_signal    = "SIGINT"
    }

    task "postgres" {
      driver = "exec"
      config {
        flake   = "github:input-output-hk/cardano-db-sync?rev=af6f4d31d137388aa59bae10c2fa79c219ce433d#postgres"
        command = "/bin/postgres-entrypoint"
      }
      env = {
        PATH   = "/bin"
        PGDATA = "/persist/postgres"
      }
      resources {
        cpu    = 2500
        memory = 1024
      }
      kill_timeout = "1m0s"
      volume_mount {
        volume           = "persist"
        destination      = "/persist"
        read_only        = false
        propagation_mode = "private"
      }
      leader         = false
      shutdown_delay = "0s"
      kill_signal    = "SIGINT"
    }

    task "cardano-node" {
      driver = "exec"
      config {
        flake   = "github:input-output-hk/cardano-node?rev=14229feb119cc3431515dde909a07bbf214f5e26#cardano-node-mainnet-debug"
        command = "/bin/cardano-node-entrypoint"
      }
      env = {
        PATH = "/bin"
      }
      resources {
        cpu    = 3600
        memory = 8192
      }
      volume_mount {
        volume           = "persist"
        destination      = "/persist"
        read_only        = false
        propagation_mode = "private"
      }
      leader         = false
      shutdown_delay = "0s"
      kill_signal    = "SIGINT"
    }

    task "snapshot" {
      driver = "exec"
      config {
        flake   = "github:input-output-hk/vit-testing/32d849099791a014902d4ff7dd8eb192afd868d#snapshot-trigger-service"
        command = "/bin/snapshot-trigger-service"
        args    = ["--config", "/secrets/snapshot.config"]
      }
      resources {
        cpu    = 1500
        memory = 2048
      }
      vault {
        policies      = ["nomad-cluster"]
        namespace     = ""
        env           = true
        change_mode   = "noop"
        change_signal = ""
      }
      template {
        source          = ""
        destination     = "genesis-template.json"
        data            = "{}"
        change_mode     = "restart"
        change_signal   = ""
        splay           = "5s"
        perms           = "0644"
        left_delimiter  = "{{"
        right_delimiter = "}}"
        env             = false
      }
      template {
        source          = ""
        destination     = "secrets/snapshot.config"
        data            = "{\n  \"port\": {{ env \"NOMAD_PORT_snapshot\" }},\n  \"result-dir\": \"/persist/snapshot\",\n  \"voting-tools\": {\n    \"bin\": \"voting-tools\",\n    \"network\": \"mainnet\",\n    \"db\": \"cexplorer\",\n    \"db-user\": \"cexplorer\",\n    \"db-host\": \"/alloc\",\n    \"scale\": 1000000\n  },\n  \"token\": \"{{with secret \"kv/data/nomad-cluster/catalyst-sync/mainnet/snapshot\"}}{{.Data.data.token}}{{end}}\"\n}"
        change_mode     = "noop"
        change_signal   = ""
        splay           = "5s"
        perms           = "0644"
        left_delimiter  = "{{"
        right_delimiter = "}}"
        env             = false
      }
      volume_mount {
        volume           = "persist"
        destination      = "/persist"
        read_only        = false
        propagation_mode = "private"
      }
      leader         = false
      shutdown_delay = "0s"
      kill_signal    = "SIGINT"
    }

    task "registration" {
      driver = "exec"
      config {
        flake   = "github:input-output-hk/vit-testing/32d849099791a014902d4ff7dd8eb192afd868d8#registration-service"
        command = "/bin/registration-service"
        args    = ["--config", "/secrets/registration.config"]
      }
      env = {
        CARDANO_NODE_SOCKET_PATH = "/alloc/node.socket"
      }
      resources {
        cpu    = 1500
        memory = 2048
      }
      vault {
        policies      = ["nomad-cluster"]
        namespace     = ""
        env           = true
        change_mode   = "noop"
        change_signal = ""
      }
      template {
        source          = ""
        destination     = "secrets/registration.config"
        data            = "{\n  \"port\": {{ env \"NOMAD_PORT_registration\" }},\n  \"jcli\": \"jcli\",\n  \"result-dir\": \"/persist/registration\",\n  \"cardano-cli\": \"cardano-cli\",\n  \"voter-registration\": \"voter-registration\",\n  \"vit-kedqr\": \"vit-kedqr\",\n  \"network\": \"mainnet\",\n  \"token\": \"{{with secret \"kv/data/nomad-cluster/catalyst-sync/mainnet/registration\"}}{{.Data.data.token}}{{end}}\"\n}"
        change_mode     = "noop"
        change_signal   = ""
        splay           = "5s"
        perms           = "0644"
        left_delimiter  = "{{"
        right_delimiter = "}}"
        env             = false
      }
      volume_mount {
        volume           = "persist"
        destination      = "/persist"
        read_only        = false
        propagation_mode = "private"
      }
      leader         = false
      shutdown_delay = "0s"
      kill_signal    = "SIGINT"
    }

    task "registration-verify" {
      driver = "exec"
      config {
        command = "/bin/registration-verify-service"
        args    = ["--config", "/secrets/registration.config"]
        flake   = "github:input-output-hk/vit-testing/32d849099791a014902d4ff7dd8eb192afd868d8#registration-verify-service"
      }
      env = {
        CARDANO_NODE_SOCKET_PATH = "/alloc/node.socket"
      }
      resources {
        cpu    = 1500
        memory = 2048
      }
      vault {
        policies      = ["nomad-cluster"]
        namespace     = ""
        env           = true
        change_mode   = "noop"
        change_signal = ""
      }
      template {
        source          = ""
        destination     = "secrets/registration.config"
        data            = "{\n  \"port\": {{ env \"NOMAD_PORT_registration_verify\" }},\n  \"jcli\": \"jcli\",\n  \"snapshot-address\": \"https://snapshot-mainnet.vit.iohk.io\",\n  \"snapshot-token\": \"{{with secret \"kv/data/nomad-cluster/catalyst-sync/mainnet/snapshot\"}}{{.Data.data.token}}{{end}}\"\n}"
        change_mode     = "noop"
        change_signal   = ""
        splay           = "5s"
        perms           = "0644"
        left_delimiter  = "{{"
        right_delimiter = "}}"
        env             = false
      }
      volume_mount {
        volume           = "persist"
        destination      = "/persist"
        read_only        = false
        propagation_mode = "private"
      }
      leader         = false
      shutdown_delay = "0s"
      kill_signal    = "SIGINT"
    }

    task "promtail" {
      driver = "exec"
      config {
        flake   = "github:NixOS/nixpkgs/nixos-21.05#grafana-loki"
        command = "/bin/promtail"
        args    = ["-config.file", "local/config.yaml"]
      }
      resources {
        cpu    = 100
        memory = 100
      }
      template {
        source          = ""
        destination     = "local/config.yaml"
        data            = "server:\n  http_listen_port: 0\n  grpc_listen_port: 0\npositions:\n  filename: /local/positions.yaml\nclient:\n  url: http://172.16.0.20:3100/loki/api/v1/push\nscrape_configs:\n- job_name: '{{ env \"NOMAD_JOB_NAME\" }}-{{ env \"NOMAD_ALLOC_INDEX\" }}'\n  pipeline_stages: null\n  static_configs:\n  - labels:\n      nomad_alloc_id: '{{ env \"NOMAD_ALLOC_ID\" }}'\n      nomad_alloc_index: '{{ env \"NOMAD_ALLOC_INDEX\" }}'\n      nomad_alloc_name: '{{ env \"NOMAD_ALLOC_NAME\" }}'\n      nomad_dc: '{{ env \"NOMAD_DC\" }}'\n      nomad_group_name: '{{ env \"NOMAD_GROUP_NAME\" }}'\n      nomad_job_id: '{{ env \"NOMAD_JOB_ID\" }}'\n      nomad_job_name: '{{ env \"NOMAD_JOB_NAME\" }}'\n      nomad_job_parent_id: '{{ env \"NOMAD_JOB_PARENT_ID\" }}'\n      nomad_namespace: '{{ env \"NOMAD_NAMESPACE\" }}'\n      nomad_region: '{{ env \"NOMAD_REGION\" }}'\n      __path__: /alloc/logs/*.std*.[0-9]*\n"
        change_mode     = "restart"
        change_signal   = ""
        splay           = "5s"
        perms           = "0644"
        left_delimiter  = "{{"
        right_delimiter = "}}"
        env             = false
      }
      leader         = false
      shutdown_delay = "0s"
      kill_signal    = "SIGINT"
    }
    volume "persist" {
      type      = "host"
      source    = "catalyst-sync-mainnet"
      read_only = false
      per_alloc = false
    }
    reschedule {
      attempts       = 0
      interval       = "0s"
      delay          = "30s"
      delay_function = "exponential"
      max_delay      = "1h0m0s"
      unlimited      = true
    }

    network {
      mode = "host"
      port "snapshot" {
      }
      port "registration" {
      }
      port "registration_verify" {
      }
    }

    service {
      name                = "catalyst-sync-snapshot-mainnet"
      tags                = ["ingress", "snapshot", "mainnet", "catalyst-sync", "traefik.enable=true", "traefik.http.routers.catalyst-sync-snapshot-mainnet.rule=Host(`snapshot-mainnet.vit.iohk.io`)", "traefik.http.routers.catalyst-sync-snapshot-mainnet.entrypoints=https", "traefik.http.routers.catalyst-sync-snapshot-mainnet.tls=true"]
      enable_tag_override = false
      port                = "snapshot"
      address_mode        = "host"
      task                = "snapshot"
    }

    service {
      name                = "catalyst-sync-registration-mainnet"
      tags                = ["ingress", "registration", "mainnet", "catalyst-sync", "traefik.enable=true", "traefik.http.routers.catalyst-sync-registration-mainnet.rule=Host(`registration-mainnet.vit.iohk.io`)", "traefik.http.routers.catalyst-sync-registration-mainnet.entrypoints=https", "traefik.http.routers.catalyst-sync-registration-mainnet.tls=true"]
      enable_tag_override = false
      port                = "registration"
      address_mode        = "host"
      task                = "registration"
    }

    service {
      name                = "catalyst-sync-registration-verify-mainnet"
      tags                = ["ingress", "registration-verify", "mainnet", "catalyst-sync", "traefik.enable=true", "traefik.http.routers.catalyst-sync-registration-verify-mainnet.rule=Host(`registration-verify-mainnet.vit.iohk.io`)", "traefik.http.routers.catalyst-sync-registration-verify-mainnet.entrypoints=https", "traefik.http.routers.catalyst-sync-registration-verify-mainnet.tls=true"]
      enable_tag_override = false
      port                = "registration_verify"
      address_mode        = "host"
      task                = "registration-verify"
    }
    shutdown_delay = "0s"
  }
  update {
    stagger           = "30s"
    max_parallel      = 1
    health_check      = "checks"
    min_healthy_time  = "10s"
    healthy_deadline  = "5m0s"
    progress_deadline = "10m0s"
    canary            = 0
    auto_revert       = false
    auto_promote      = false
  }
}
