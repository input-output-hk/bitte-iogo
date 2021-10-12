package main

import (
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/require"
)

func TestJob2Hcl(t *testing.T) {
	r := require.New(t)

	compare(r, "fixtures/1.hcl", &api.Job{
		Name:        ptrStr("mysql"),
		Region:      ptrStr("eu"),
		Namespace:   ptrStr("test"),
		ID:          ptrStr("123"),
		Type:        ptrStr("batch"),
		Priority:    ptrInt(50),
		AllAtOnce:   ptrBool(true),
		Datacenters: []string{"a", "b"},
		Constraints: []*api.Constraint{
			{
				LTarget: "left",
				RTarget: "right",
				Operand: "op",
			},
		},
		Affinities: []*api.Affinity{
			{
				LTarget: "left",
				RTarget: "right",
				Operand: "op",
				Weight:  ptrInt8(10),
			},
		},
		TaskGroups: []*api.TaskGroup{
			{
				Name: ptrStr("mysqld"),
				RestartPolicy: &api.RestartPolicy{
					Attempts: ptrInt(3),
					Delay:    ptrDuration(10 * time.Second),
					Interval: ptrDuration(10 * time.Minute),
					Mode:     ptrStr("fail"),
				},
				Tasks: []*api.Task{{
					Name: "server",
					Services: []*api.Service{{
						Tags:      []string{"leader", "mysql"},
						PortLabel: "db",
						Checks: []api.ServiceCheck{
							{Type: "tcp", PortLabel: "db", Interval: 10 * time.Second, Timeout: 2 * time.Second},
							{Type: "script", Name: "check_table",
								Command:  "/usr/local/bin/check_mysql_table_status",
								Args:     []string{"--verbose"},
								Interval: 60 * time.Second, Timeout: 5 * time.Second,
								CheckRestart: &api.CheckRestart{
									Limit:          3,
									Grace:          ptrDuration(90 * time.Second),
									IgnoreWarnings: false,
								},
							},
						},
					}},
				}},
			},
		},
		Meta: map[string]string{"hi": "there"},
	})

	compare(r, "fixtures/2.hcl", &api.Job{
		Name: ptrStr("docs"),
		TaskGroups: []*api.TaskGroup{
			{
				Name: ptrStr("example"),
				Tasks: []*api.Task{{
					Name: "server",
					Artifacts: []*api.TaskArtifact{
						{
							GetterSource: ptrStr("https://example.com/file.tar.gz"),
							RelativeDest: ptrStr("local/some-directory"),
							GetterOptions: map[string]string{
								"checksum": "md5:df6a4178aec9fbdc1d6d7e3634d1bc33",
								"depth":    "1",
							},
							GetterHeaders: map[string]string{
								"User-Agent":    "nomad-[${NOMAD_JOB_ID}]-[${NOMAD_GROUP_NAME}]-[${NOMAD_TASK_NAME}]",
								"X-Nomad-Alloc": "${NOMAD_ALLOC_ID}",
							},
						},
					},
				}},
			},
		},
	})

	compare(r, "fixtures/3.hcl", &api.Job{
		Name: ptrStr("docs"),

		Affinities: []*api.Affinity{{
			LTarget: "${node.datacenter}",
			RTarget: "us-west1",
			Weight:  ptrInt8(100),
		}},

		TaskGroups: []*api.TaskGroup{{
			Name: ptrStr("example"),

			Affinities: []*api.Affinity{{
				LTarget: "${meta.rack}",
				RTarget: "r1",
				Weight:  ptrInt8(50),
			}},

			Tasks: []*api.Task{{
				Name: "server",

				Affinities: []*api.Affinity{{
					LTarget: "${meta.my_custom_value}",
					RTarget: "3",
					Operand: ">",
					Weight:  ptrInt8(50),
				}},
			}},
		}},
	})

	compare(r, "fixtures/4.hcl", &api.Job{
		Name:        ptrStr("countdash"),
		Datacenters: []string{"dc1"},

		TaskGroups: []*api.TaskGroup{
			{
				Name: ptrStr("api"),

				Networks: []*api.NetworkResource{{
					Mode: "bridge",
				}},

				Services: []*api.Service{{
					Name:      "count-api",
					PortLabel: "9001",

					Connect: &api.ConsulConnect{
						SidecarService: &api.ConsulSidecarService{},
					},

					Checks: []api.ServiceCheck{{
						Expose:   true,
						Type:     "http",
						Name:     "api-health",
						Path:     "/health",
						Interval: 10 * time.Second,
						Timeout:  3 * time.Second,
					}},
				}},

				Tasks: []*api.Task{{
					Name:   "web",
					Driver: "docker",
					Config: map[string]interface{}{
						"image": "hashicorpnomad/counter-api:v3",
					},
				}},
			},

			{
				Name: ptrStr("dashboard"),

				Networks: []*api.NetworkResource{{
					Mode: "bridge",
					DynamicPorts: []api.Port{{
						Label: "http",
						Value: 9002,
						To:    9002,
					}},
				}},

				Services: []*api.Service{{
					Name:      "count-dashboard",
					PortLabel: "9002",

					Connect: &api.ConsulConnect{
						SidecarService: &api.ConsulSidecarService{
							Proxy: &api.ConsulProxy{
								Upstreams: []*api.ConsulUpstream{{
									DestinationName: "count-api",
									LocalBindPort:   8080,
								}},
							},
						},
					},
				}},

				Tasks: []*api.Task{{
					Name:   "dashboard",
					Driver: "docker",

					Config: map[string]interface{}{
						"image": "hashicorpnomad/counter-dashboard:v3",
					},

					Env: map[string]string{
						"COUNTING_SERVICE_URL": "http://${NOMAD_UPSTREAM_ADDR_count_api}",
					},
				}},
			},
		},
	})

	compare(r, "fixtures/5.hcl", &api.Job{
		Name: ptrStr("docs"),
		Constraints: []*api.Constraint{{
			LTarget: "${attr.kernel.name}",
			RTarget: "linux",
		}},
		TaskGroups: []*api.TaskGroup{{
			Name: ptrStr("example"),
			Constraints: []*api.Constraint{{
				Operand: "distinct_hosts",
				RTarget: "true",
			}},
			Tasks: []*api.Task{{
				Name: "server",
				Constraints: []*api.Constraint{{
					LTarget: "${meta.my_custom_value}",
					RTarget: "3",
					Operand: ">",
				}},
			}},
		}},
	})

	compare(r, "fixtures/6.hcl", &api.Job{
		Name: ptrStr("docs"),
		TaskGroups: []*api.TaskGroup{{
			Name: ptrStr("example"),
			Tasks: []*api.Task{{
				Name: "server",
				Resources: &api.Resources{
					Devices: []*api.RequestedDevice{{
						Name:  "nvidia/gpu",
						Count: ptrUInt64(2),
						Constraints: []*api.Constraint{{
							LTarget: "${device.attr.memory}",
							Operand: ">=",
							RTarget: "2 GiB",
						}},
						Affinities: []*api.Affinity{{
							LTarget: "${device.attr.memory}",
							RTarget: "4 GiB",
							Operand: ">=",
							Weight:  ptrInt8(75),
						}},
					}},
				},
			}},
		}},
	})

	compare(r, "fixtures/7.hcl", &api.Job{
		Name: ptrStr("docs"),
		TaskGroups: []*api.TaskGroup{{
			Name: ptrStr("example"),
			Tasks: []*api.Task{{
				Name:            "server",
				DispatchPayload: &api.DispatchPayloadConfig{File: "config.json"},
			}},
		}},
	})

	compare(r, "fixtures/8.hcl", &api.Job{
		Name: ptrStr("docs"),
		TaskGroups: []*api.TaskGroup{{
			Name: ptrStr("example"),
			EphemeralDisk: &api.EphemeralDisk{
				Migrate: ptrBool(true),
				SizeMB:  ptrInt(500),
				Sticky:  ptrBool(true),
			},
		}},
	})

	compare(r, "fixtures/9.hcl", &api.Job{
		Name:        ptrStr("ingress-demo"),
		Datacenters: []string{"dc1"},
		TaskGroups: []*api.TaskGroup{{
			Name: ptrStr("ingress-group"),
			Networks: []*api.NetworkResource{{
				Mode: "bridge",
				DynamicPorts: []api.Port{{
					Label: "inbound",
					Value: 8080,
					To:    8080,
				}},
			}},
			Services: []*api.Service{{
				Name:      "my-ingress-service",
				PortLabel: "8080",
				Connect: &api.ConsulConnect{
					Gateway: &api.ConsulGateway{
						Proxy: &api.ConsulGatewayProxy{
							EnvoyGatewayNoDefaultBind: true,
							// TODO: there's not a whole lot of type information to handle this...
							EnvoyGatewayBindAddresses: map[string]*api.ConsulGatewayBindAddress{
								"address": {Name: "address", Address: "0.0.0.0", Port: 12},
							},
						},
						Ingress: &api.ConsulIngressConfigEntry{
							Listeners: []*api.ConsulIngressListener{{
								Port:     8080,
								Protocol: "tcp",
								Services: []*api.ConsulIngressService{{
									Name: "uuid-api",
								}},
							}},
						},
					},
				},
			}},
		},
			{
				Name: ptrStr("generator"),
				Networks: []*api.NetworkResource{{
					Mode: "host",
					DynamicPorts: []api.Port{{
						Label: "api",
					}},
				}},

				Services: []*api.Service{{
					Name:      "uuid-api",
					PortLabel: "api",
					Connect: &api.ConsulConnect{
						Native: true,
					},
				}},

				Tasks: []*api.Task{{
					Name:   "generate",
					Driver: "docker",
					Config: map[string]interface{}{
						"image":        "hashicorpnomad/uuid-api:v5",
						"network_mode": "host",
					},
					Env: map[string]string{
						"BIND": "0.0.0.0",
						"PORT": "${NOMAD_PORT_api}",
					},
				}},
			}},
	})

	compare(r, "fixtures/10.hcl", &api.Job{
		Name: ptrStr("docs"),
		TaskGroups: []*api.TaskGroup{{
			Name: ptrStr("example"),
			Volumes: map[string]*api.VolumeRequest{
				"certs": {
					Type:     "host",
					ReadOnly: true,
					Source:   "ca-certificates",
				}},
			Tasks: []*api.Task{{
				Name: "example",
				VolumeMounts: []*api.VolumeMount{{
					Volume:      ptrStr("certs"),
					Destination: ptrStr("/etc/ssl/certs"),
				}},
			}},
		}},
	})
}

func compare(r *require.Assertions, fixturePath string, job *api.Job) {
	b, err := ioutil.ReadFile(fixturePath)
	r.Nil(err)
	f, err := any2hcl("job", job)
	r.Nil(err)
	r.Equal(strings.TrimSpace(string(b)), strings.TrimSpace(string(f.Bytes())))
}

func ptrStr(v string) *string {
	return &v
}

func ptrInt(v int) *int {
	return &v
}

func ptrInt8(v int8) *int8 {
	return &v
}

func ptrUInt64(v uint64) *uint64 {
	return &v
}

func ptrDuration(v time.Duration) *time.Duration {
	return &v
}

func ptrBool(v bool) *bool {
	return &v
}
