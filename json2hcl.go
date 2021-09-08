package main

import (
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/nomad/api"
	"github.com/zclconf/go-cty/cty"
)

func job2hcl(job *api.Job) *hclwrite.File {
	f := hclwrite.NewEmptyFile()
	body := f.Body()

	encodeJob(body, job)

	return f
}

func encodeJob(parent *hclwrite.Body, job *api.Job) {
	body := parent.AppendNewBlock("job", []string{*job.Name}).Body()

	setAttributeValue(body, "region", job.Region)
	setAttributeValue(body, "namespace", job.Namespace)
	setAttributeValue(body, "id", job.ID)
	setAttributeValue(body, "type", job.Type)
	setAttributeValue(body, "priority", job.Priority)
	setAttributeValue(body, "all_at_once", job.AllAtOnce)
	setAttributeValue(body, "datacenters", job.Datacenters)

	setConstraints(body, job.Constraints)
	setAffinities(body, job.Affinities)
	setTaskGroups(body, job.TaskGroups)

	setUpdateStrategy(body, job.Update)
	// setMultiregion(body, job.Multiregion) // Enterprise only
	setSpreads(body, job.Spreads)
	setPeriodicConfig(body, job.Periodic)
	setParameterizedJobConfig(body, job.ParameterizedJob)
	setReschedulePolicy(body, job.Reschedule)
	setMigrateStrategy(body, job.Migrate)
	setMeta(body, job.Meta)
	setAttributeValue(body, "consul_token", job.ConsulToken)
	setAttributeValue(body, "vault_token", job.VaultToken)
}

func setMeta(parent *hclwrite.Body, meta map[string]string) {
	if len(meta) == 0 {
		return
	}

	body := parent.AppendNewBlock("meta", nil).Body()
	for k, v := range meta {
		setAttributeValue(body, k, v)
	}
}

func setParameterizedJobConfig(parent *hclwrite.Body, parameterized *api.ParameterizedJobConfig) {
	if parameterized == nil {
		return
	}

	body := parent.AppendNewBlock("parameterized", nil).Body()
	setAttributeValue(body, "payload", parameterized.Payload)
	setAttributeValue(body, "meta_required", parameterized.MetaRequired)
	setAttributeValue(body, "meta_optional", parameterized.MetaOptional)
}

func setMigrateStrategy(parent *hclwrite.Body, migrate *api.MigrateStrategy) {
	if migrate == nil {
		return
	}

	body := parent.AppendNewBlock("migrate", nil).Body()
	setAttributeValue(body, "max_parallel", migrate.MaxParallel)
	setAttributeValue(body, "health_check", migrate.HealthCheck)
	setAttributeValue(body, "min_healthy_time", migrate.MinHealthyTime)
	setAttributeValue(body, "healthy_deadline", migrate.HealthyDeadline)
}

func setPeriodicConfig(parent *hclwrite.Body, config *api.PeriodicConfig) {
	if config == nil {
		return
	}

	body := parent.AppendNewBlock("periodic", nil).Body()
	setAttributeValue(body, "enabled", config.Enabled)
	setAttributeValue(body, "cron", config.Spec)
	setAttributeValue(body, "prohibit_overlap", config.ProhibitOverlap)
	setAttributeValue(body, "time_zone", config.TimeZone)
}

func setTaskGroups(parent *hclwrite.Body, taskGroups []*api.TaskGroup) {
	for _, tg := range taskGroups {
		parent.AppendNewline()

		body := parent.AppendNewBlock("group", []string{*tg.Name}).Body()
		setAttributeValue(body, "count", tg.Count)
		setConstraints(body, tg.Constraints)
		setAffinities(body, tg.Affinities)
		setTasks(body, tg.Tasks)
		setSpreads(body, tg.Spreads)
		setVolumes(body, tg.Volumes)
		setRestartPolicy(body, tg.RestartPolicy)
		setReschedulePolicy(body, tg.ReschedulePolicy)
		setEphemeralDisk(body, tg.EphemeralDisk)
		setUpdateStrategy(body, tg.Update)
		setMigrateStrategy(body, tg.Migrate)
		setNetworkResource(body, tg.Networks)
		setMeta(body, tg.Meta)
		setServices(body, tg.Services)
		setAttributeValue(body, "shutdown_delay", tg.ShutdownDelay)
		setAttributeValue(body, "stop_after_client_disconnect", tg.StopAfterClientDisconnect)
		setScaling(body, tg.Scaling)
		// setConsul(body, tg.Consul) Enterprise only
	}
}

func setScaling(parent *hclwrite.Body, scaling *api.ScalingPolicy) {
	if scaling == nil {
		return
	}

	body := parent.AppendNewBlock("scaling", nil).Body()
	setAttributeValue(body, "min", scaling.Min)
	setAttributeValue(body, "max", scaling.Max)

	if len(scaling.Policy) > 0 {
		panic("Cannot process scaling.policy")
	}

	setAttributeValue(body, "enabled", scaling.Enabled)
	setAttributeValue(body, "type", scaling.Type)
}

func setServices(parent *hclwrite.Body, services []*api.Service) {
	for _, service := range services {
		parent.AppendNewline()

		body := parent.AppendNewBlock("service", nil).Body()
		setAttributeValue(body, "name", service.Name)
		setAttributeValue(body, "tags", service.Tags)
		setAttributeValue(body, "canary_tags", service.CanaryTags)
		setAttributeValue(body, "enable_tag_override", service.EnableTagOverride)
		setAttributeValue(body, "port", service.PortLabel)
		setAttributeValue(body, "address_mode", service.AddressMode)
		setChecks(body, service.Checks)
		setCheckRestart(body, service.CheckRestart)
		setConnect(body, service.Connect)
		setMeta(body, service.Meta)
		setAttributeValue(body, "canary_meta", service.CanaryMeta)
		setAttributeValue(body, "task", service.TaskName)
		setAttributeValue(body, "on_update", service.OnUpdate)
	}
}

func setConnect(parent *hclwrite.Body, connect *api.ConsulConnect) {
	if connect == nil {
		return
	}

	body := parent.AppendNewBlock("connect", nil).Body()
	setAttributeValue(body, "native", connect.Native)
	setConsulGateway(body, connect.Gateway)
	setConsulSidecarService(body, connect.SidecarService)
	setSidecarTask(body, connect.SidecarTask)
}

func setConsulGateway(parent *hclwrite.Body, gateway *api.ConsulGateway) {
	if gateway == nil {
		return
	}

	body := parent.AppendNewBlock("gateway", nil).Body()
	setConsulGatewayProxy(body, gateway.Proxy)
	setConsulIngressConfigEntry(body, gateway.Ingress)
	setConsulTerminatingConfigEntry(body, gateway.Terminating)
	setConsulMeshConfigEntry(body, gateway.Mesh)
}

func setConsulMeshConfigEntry(parent *hclwrite.Body, entry *api.ConsulMeshConfigEntry) {
	if entry == nil {
		return
	}

	parent.AppendNewBlock("mesh", nil).Body()
}

func setConsulIngressConfigEntry(parent *hclwrite.Body, entry *api.ConsulIngressConfigEntry) {
	if entry == nil {
		return
	}

	body := parent.AppendNewBlock("ingress", nil).Body()
	setConsulGatewayTLSConfig(body, entry.TLS)
	setConsulIngressListeners(body, entry.Listeners)
}

func setConsulGatewayTLSConfig(parent *hclwrite.Body, tls *api.ConsulGatewayTLSConfig) {
	if tls == nil {
		return
	}

	body := parent.AppendNewBlock("tls", nil).Body()
	setAttributeValue(body, "enabled", tls.Enabled)
}

func setConsulIngressListeners(parent *hclwrite.Body, listeners []*api.ConsulIngressListener) {
	for _, listener := range listeners {
		body := parent.AppendNewBlock("listener", nil).Body()
		setAttributeValue(body, "port", listener.Port)
		setAttributeValue(body, "protocol", listener.Protocol)
		setConsulIngressServices(body, listener.Services)
	}
}

func setConsulIngressServices(parent *hclwrite.Body, services []*api.ConsulIngressService) {
	for _, service := range services {
		body := parent.AppendNewBlock("service", nil).Body()
		setAttributeValue(body, "name", service.Name)
		setAttributeValue(body, "hosts", service.Hosts)
	}
}

func setConsulTerminatingConfigEntry(parent *hclwrite.Body, entry *api.ConsulTerminatingConfigEntry) {
	if entry == nil {
		return
	}

	body := parent.AppendNewBlock("terminating", nil).Body()
	setConsulLinkedServices(body, entry.Services)
}

func setConsulLinkedServices(parent *hclwrite.Body, services []*api.ConsulLinkedService) {
	for _, service := range services {
		body := parent.AppendNewBlock("service", nil).Body()
		setAttributeValue(body, "name", service.Name)
		setAttributeValue(body, "ca_file", service.CAFile)
		setAttributeValue(body, "cert_file", service.CertFile)
		setAttributeValue(body, "key_file", service.KeyFile)
		setAttributeValue(body, "sni", service.SNI)
	}
}

func setConsulGatewayProxy(parent *hclwrite.Body, proxy *api.ConsulGatewayProxy) {
	if proxy == nil {
		return
	}

	body := parent.AppendNewBlock("proxy", nil).Body()
	setAttributeValue(body, "connect_timeout", proxy.ConnectTimeout)
	setAttributeValue(body, "envoy_gateway_bind_tagged_addresses", proxy.EnvoyGatewayBindTaggedAddresses)
	setAttributeValue(body, "envoy_gateway_bind_addresses", proxy.EnvoyGatewayBindTaggedAddresses)
	setAttributeValue(body, "envoy_gateway_no_default_bind", proxy.EnvoyGatewayNoDefaultBind)
	setAttributeValue(body, "envoy_dns_discovery_type", proxy.EnvoyDNSDiscoveryType)
	setAttributeValue(body, "config", proxy.Config)
}

func setConsulSidecarService(parent *hclwrite.Body, service *api.ConsulSidecarService) {
	if service == nil {
		return
	}

	body := parent.AppendNewBlock("sidecar_service", nil).Body()
	setAttributeValue(body, "tags", service.Tags)
	setAttributeValue(body, "port", service.Port)
	setConsulProxy(body, service.Proxy)
	setAttributeValue(body, "disable_default_tcp_check", service.DisableDefaultTCPCheck)
}

func setConsulProxy(parent *hclwrite.Body, proxy *api.ConsulProxy) {
	if proxy == nil {
		return
	}

	body := parent.AppendNewBlock("proxy", nil).Body()
	setAttributeValue(body, "local_service_address", proxy.LocalServiceAddress)
	setAttributeValue(body, "local_service_port", proxy.LocalServicePort)
	setConsulExposeConfig(body, proxy.ExposeConfig)
	setConsulUpstreams(body, proxy.Upstreams)
	setAttributeValue(body, "config", proxy.Config)
}

func setConsulUpstreams(parent *hclwrite.Body, upstreams []*api.ConsulUpstream) {
	for _, upstream := range upstreams {
		body := parent.AppendNewBlock("upstreams", nil).Body()
		setAttributeValue(body, "destination_name", upstream.DestinationName)
		setAttributeValue(body, "local_bind_port", upstream.LocalBindPort)
		setAttributeValue(body, "datacenter", upstream.Datacenter)
		setAttributeValue(body, "local_bind_address", upstream.Datacenter)
		setConsulMeshGateway(body, upstream.MeshGateway)
	}
}

func setConsulMeshGateway(parent *hclwrite.Body, gateway *api.ConsulMeshGateway) {
	if gateway == nil {
		return
	}

	body := parent.AppendNewBlock("mesh_gateway", nil).Body()
	setAttributeValue(body, "mode", gateway.Mode)
}

func setConsulExposeConfig(parent *hclwrite.Body, config *api.ConsulExposeConfig) {
	if config == nil {
		return
	}

	body := parent.AppendNewBlock("expose", nil).Body()
	setConsulExposePaths(body, config.Path)
}

// Path []*ConsulExposePath `mapstructure:"path" hcl:"path,block"`
func setConsulExposePaths(parent *hclwrite.Body, paths []*api.ConsulExposePath) {
	for _, path := range paths {
		body := parent.AppendNewBlock("path", nil).Body()
		setAttributeValue(body, "path", path.Path)
		setAttributeValue(body, "protocol", path.Protocol)
		setAttributeValue(body, "local_path_port", path.LocalPathPort)
		setAttributeValue(body, "listener_port", path.ListenerPort)
	}
}

func setSidecarTask(parent *hclwrite.Body, task *api.SidecarTask) {
	if task == nil {
		return
	}

	body := parent.AppendNewBlock("sidecar_task", nil).Body()

	setAttributeValue(body, "name", task.Name)
	setAttributeValue(body, "driver", task.Driver)
	setAttributeValue(body, "user", task.User)
	setAttributeValue(body, "config", task.Config)
	setAttributeValue(body, "env", task.Env)
	setResources(body, task.Resources)
	setMeta(body, task.Meta)
	setAttributeValue(body, "kill_timeout", task.KillTimeout)
	setLogConfig(body, task.LogConfig)
	setAttributeValue(body, "shutdown_delay", task.ShutdownDelay)
	setAttributeValue(body, "kill_signal", task.KillSignal)
}

func setResources(parent *hclwrite.Body, resources *api.Resources) {
	if resources == nil {
		return
	}

	body := parent.AppendNewBlock("resources", nil).Body()
	setAttributeValue(body, "cpu", resources.CPU)
	setAttributeValue(body, "cores", resources.Cores)
	setAttributeValue(body, "memory", resources.MemoryMB)
	setAttributeValue(body, "memory_max", resources.MemoryMaxMB)
	setAttributeValue(body, "disk", resources.DiskMB)
	setNetworkResource(body, resources.Networks)
	setRequestedDevices(body, resources.Devices)
}

func setRequestedDevices(parent *hclwrite.Body, devices []*api.RequestedDevice) {
	for _, device := range devices {
		parent.AppendNewline()

		body := parent.AppendNewBlock("device", []string{device.Name}).Body()
		setAttributeValue(body, "count", device.Count)
		setConstraints(body, device.Constraints)
		setAffinities(body, device.Affinities)
	}
}

func setLogConfig(parent *hclwrite.Body, config *api.LogConfig) {
	if config == nil {
		return
	}

	body := parent.AppendNewBlock("logs", nil).Body()
	setAttributeValue(body, "max_files", config.MaxFiles)
	setAttributeValue(body, "max_file_size", config.MaxFileSizeMB)
}

func setChecks(parent *hclwrite.Body, checks []api.ServiceCheck) {
	for _, check := range checks {
		parent.AppendNewline()

		body := parent.AppendNewBlock("check", nil).Body()
		setAttributeValue(body, "name", check.Name)
		setAttributeValue(body, "type", check.Type)
		setAttributeValue(body, "command", check.Command)
		setAttributeValue(body, "args", check.Args)
		setAttributeValue(body, "path", check.Path)
		setAttributeValue(body, "protocol", check.Protocol)
		setAttributeValue(body, "port", check.PortLabel)
		setAttributeValue(body, "expose", check.Expose)
		setAttributeValue(body, "address_mode", check.AddressMode)
		setAttributeValue(body, "interval", check.Interval)
		setAttributeValue(body, "timeout", check.Timeout)
		setAttributeValue(body, "initial_status", check.InitialStatus)
		setAttributeValue(body, "tls_skip_verify", check.TLSSkipVerify)
		setAttributeValue(body, "header", check.Header)
		setAttributeValue(body, "method", check.Method)
		setCheckRestart(body, check.CheckRestart)
		setAttributeValue(body, "grpc_service", check.GRPCService)
		setAttributeValue(body, "grpc_use_tls", check.GRPCUseTLS)
		setAttributeValue(body, "task", check.TaskName)
		setAttributeValue(body, "success_before_passing", check.SuccessBeforePassing)
		setAttributeValue(body, "failures_before_critical", check.FailuresBeforeCritical)
		setAttributeValue(body, "body", check.Body)
		setAttributeValue(body, "on_update", check.OnUpdate)
	}
}

func setCheckRestart(parent *hclwrite.Body, restart *api.CheckRestart) {
	if restart == nil {
		return
	}

	body := parent.AppendNewBlock("check_restart", nil).Body()
	setAttributeValue(body, "limit", restart.Limit)
	setAttributeValue(body, "grace", restart.Grace)
	setAttributeValue(body, "ignore_warnings", restart.IgnoreWarnings)
}

func setNetworkResource(parent *hclwrite.Body, networks []*api.NetworkResource) {
	for _, network := range networks {
		parent.AppendNewline()

		body := parent.AppendNewBlock("network", nil).Body()
		setAttributeValue(body, "mode", network.Mode)
		setAttributeValue(body, "device", network.Device)
		setAttributeValue(body, "cidr", network.CIDR)
		setAttributeValue(body, "ip", network.IP)
		setAttributeValue(body, "ip", network.IP)
		setDNSConfig(body, network.DNS)
		setPorts(body, network.ReservedPorts)
		setPorts(body, network.DynamicPorts)
	}
}

func setPorts(parent *hclwrite.Body, ports []api.Port) {
	for _, port := range ports {
		body := parent.AppendNewBlock("port", []string{port.Label}).Body()
		setAttributeValue(body, "host_network", port.HostNetwork)
		if port.To != 0 {
			setAttributeValue(body, "to", port.To)
		}
		if port.Value != 0 {
			setAttributeValue(body, "static", port.Value)
		}
	}
}

func setDNSConfig(parent *hclwrite.Body, dns *api.DNSConfig) {
	if dns == nil {
		return
	}

	body := parent.AppendNewBlock("dns", nil).Body()
	gohcl.EncodeIntoBody(dns, body)
}

func setUpdateStrategy(parent *hclwrite.Body, update *api.UpdateStrategy) {
	if update == nil {
		return
	}

	body := parent.AppendNewBlock("update", nil).Body()
	setAttributeValue(body, "stagger", update.Stagger)
	setAttributeValue(body, "max_parallel", update.MaxParallel)
	setAttributeValue(body, "health_check", update.HealthCheck)
	setAttributeValue(body, "min_healthy_time", update.MinHealthyTime)
	setAttributeValue(body, "healthy_deadline", update.HealthyDeadline)
	setAttributeValue(body, "progress_deadline", update.ProgressDeadline)
	setAttributeValue(body, "canary", update.Canary)
	setAttributeValue(body, "auto_revert", update.AutoRevert)
	setAttributeValue(body, "auto_promote", update.AutoPromote)
}

func setEphemeralDisk(parent *hclwrite.Body, disk *api.EphemeralDisk) {
	if disk == nil {
		return
	}

	body := parent.AppendNewBlock("ephemeral_disk", nil).Body()
	gohcl.EncodeIntoBody(disk, body)
}

func setReschedulePolicy(parent *hclwrite.Body, policy *api.ReschedulePolicy) {
	if policy == nil {
		return
	}

	body := parent.AppendNewBlock("reschedule", nil).Body()
	setAttributeValue(body, "attempts", policy.Attempts)
	setAttributeValue(body, "interval", policy.Interval)
	setAttributeValue(body, "delay", policy.Delay)
	setAttributeValue(body, "delay_function", policy.DelayFunction)
	setAttributeValue(body, "max_delay", policy.MaxDelay)
	setAttributeValue(body, "unlimited", policy.Unlimited)
}

func setRestartPolicy(parent *hclwrite.Body, policy *api.RestartPolicy) {
	if policy == nil {
		return
	}

	body := parent.AppendNewBlock("restart", nil).Body()
	setAttributeValue(body, "interval", policy.Interval)
	setAttributeValue(body, "attempts", policy.Attempts)
	setAttributeValue(body, "delay", policy.Delay)
	setAttributeValue(body, "mode", policy.Mode)
}

func setVolumes(parent *hclwrite.Body, volumes map[string]*api.VolumeRequest) {
	for name, vr := range volumes {
		body := parent.AppendNewBlock("volume", []string{name}).Body()
		setAttributeValue(body, "type", vr.Type)
		setAttributeValue(body, "source", vr.Source)
		setAttributeValue(body, "read_only", vr.ReadOnly)
		setAttributeValue(body, "access_mode", vr.AccessMode)
		setAttributeValue(body, "attachment_mode", vr.AttachmentMode)
		setAttributeValue(body, "per_alloc", vr.PerAlloc)
		setMountOptions(body, vr.MountOptions)
	}
}

func setMountOptions(parent *hclwrite.Body, options *api.CSIMountOptions) {
	if options == nil {
		return
	}

	body := parent.AppendNewBlock("mount_options", nil).Body()
	gohcl.EncodeIntoBody(options, body)
}

func setSpreads(parent *hclwrite.Body, spreads []*api.Spread) {
	for _, spread := range spreads {
		body := parent.AppendNewBlock("spread", nil).Body()
		setAttributeValue(body, "attribute", spread.Attribute)
		setSpreadTarget(body, spread.SpreadTarget)
	}
}

func setSpreadTarget(parent *hclwrite.Body, sts []*api.SpreadTarget) {
	for _, st := range sts {
		body := parent.AppendNewBlock("target", []string{st.Value}).Body()
		setAttributeValue(body, "percent", st.Percent)
	}
}

func setTaskConfig(parent *hclwrite.Body, config map[string]interface{}) {
	if len(config) == 0 {
		return
	}

	body := parent.AppendNewBlock("config", nil).Body()

	for mk, mv := range config {
		setAttributeValue(body, mk, mv)
	}
}

func setTasks(parent *hclwrite.Body, tasks []*api.Task) {
	for _, task := range tasks {
		parent.AppendNewline()

		body := parent.AppendNewBlock("task", []string{task.Name}).Body()
		setAttributeValue(body, "driver", task.Driver)
		setAttributeValue(body, "user", task.User)
		setTaskLifecycle(body, task.Lifecycle)
		setTaskConfig(body, task.Config)
		setConstraints(body, task.Constraints)
		setAffinities(body, task.Affinities)
		setAttributeValue(body, "env", task.Env)
		setServices(body, task.Services)
		setResources(body, task.Resources)
		setRestartPolicy(body, task.RestartPolicy)
		setMeta(body, task.Meta)
		setAttributeValue(body, "kill_timeout", task.KillTimeout)
		setLogConfig(body, task.LogConfig)
		setTaskArtifacts(body, task.Artifacts)
		setVault(body, task.Vault)
		setTemplates(body, task.Templates)
		setDispatchPayloadConfig(body, task.DispatchPayload)
		setVolumeMounts(body, task.VolumeMounts)
		setCSIPluginConfig(body, task.CSIPluginConfig)
		setAttributeValue(body, "leader", task.Leader)
		setAttributeValue(body, "shutdown_delay", task.ShutdownDelay)
		setAttributeValue(body, "kill_signal", task.KillSignal)
		setAttributeValue(body, "kind", task.Kind)
		setScalingPolicies(body, task.ScalingPolicies)
	}
}

func setCSIPluginConfig(parent *hclwrite.Body, config *api.TaskCSIPluginConfig) {
	if config == nil {
		return
	}

	body := parent.AppendNewBlock("csi_plugin", nil).Body()
	setAttributeValue(body, "id", config.ID)
	setAttributeValue(body, "type", config.Type)
	setAttributeValue(body, "mount_dir", config.MountDir)
}

func setScalingPolicies(parent *hclwrite.Body, policies []*api.ScalingPolicy) {
	for _, policy := range policies {
		setScaling(parent, policy)
	}
}

func setVolumeMounts(parent *hclwrite.Body, mounts []*api.VolumeMount) {
	for _, mount := range mounts {
		body := parent.AppendNewBlock("volume_mount", nil).Body()
		gohcl.EncodeIntoBody(mount, body)
	}
}

func setDispatchPayloadConfig(parent *hclwrite.Body, config *api.DispatchPayloadConfig) {
	if config == nil {
		return
	}

	body := parent.AppendNewBlock("dispatch_payload", nil).Body()
	gohcl.EncodeIntoBody(config, body)
}

func setTemplates(parent *hclwrite.Body, templates []*api.Template) {
	for _, template := range templates {
		body := parent.AppendNewBlock("template", nil).Body()
		setAttributeValue(body, "source", template.SourcePath)
		setAttributeValue(body, "destination", template.DestPath)
		setAttributeValue(body, "data", template.EmbeddedTmpl)
		setAttributeValue(body, "change_mode", template.ChangeMode)
		setAttributeValue(body, "change_signal", template.ChangeSignal)
		setAttributeValue(body, "splay", template.Splay)
		setAttributeValue(body, "perms", template.Perms)
		setAttributeValue(body, "left_delimiter", template.LeftDelim)
		setAttributeValue(body, "right_delimiter", template.RightDelim)
		setAttributeValue(body, "env", template.Envvars)
		setAttributeValue(body, "vault_grace", template.VaultGrace)
	}
}

func setVault(parent *hclwrite.Body, vault *api.Vault) {
	if vault == nil {
		return
	}

	body := parent.AppendNewBlock("vault", nil).Body()
	gohcl.EncodeIntoBody(vault, body)
}

func setTaskArtifacts(parent *hclwrite.Body, artifacts []*api.TaskArtifact) {
	for _, artifact := range artifacts {
		body := parent.AppendNewBlock("artifact", nil).Body()
		setAttributeValue(body, "source", artifact.GetterSource)
		setAttributeValue(body, "options", artifact.GetterOptions)
		setAttributeValue(body, "headers", artifact.GetterHeaders)
		setAttributeValue(body, "mode", artifact.GetterMode)
		setAttributeValue(body, "destination", artifact.RelativeDest)
	}
}

func setTaskLifecycle(parent *hclwrite.Body, lifecycle *api.TaskLifecycle) {
	if lifecycle == nil {
		return
	}

	body := parent.AppendNewBlock("lifecycle", nil).Body()
	gohcl.EncodeIntoBody(lifecycle, body)
}

func setAffinities(parent *hclwrite.Body, affinities []*api.Affinity) {
	for _, affinity := range affinities {
		body := parent.AppendNewBlock("affinity", nil).Body()
		gohcl.EncodeIntoBody(affinity, body)
	}
}

func setConstraints(parent *hclwrite.Body, constraints []*api.Constraint) {
	for _, constraint := range constraints {
		body := parent.AppendNewBlock("constraint", nil).Body()
		gohcl.EncodeIntoBody(constraint, body)
	}
}

func setAttributeValue(body *hclwrite.Body, key string, value interface{}) {
	cv, ok := convertValue(value)
	if ok {
		body.SetAttributeValue(key, cv)
	}
}

func convertValue(value interface{}) (cv cty.Value, ok bool) {
	switch v := value.(type) {
	case string:
		if v != "" {
			return cty.StringVal(v), true
		}
	case *string:
		if v != nil {
			return cty.StringVal(*v), true
		}
	case int:
		return cty.NumberIntVal(int64(v)), true
	case *int:
		if v != nil {
			return cty.NumberIntVal(int64(*v)), true
		}
	case *bool:
		if v != nil {
			return cty.BoolVal(*v), true
		}
	case bool:
		return cty.BoolVal(v), true
	case time.Duration:
		return cty.StringVal(v.String()), true
	case *time.Duration:
		if v != nil {
			return cty.StringVal(v.String()), true
		}
	case []interface{}:
		if len(v) != 0 {
			list := []cty.Value{}
			for _, value := range v {
				converted, ok := convertValue(value)
				if !ok {
					return cv, false
				}
				list = append(list, converted)
			}
			return cty.ListVal(list), true
		}
	case []string:
		if len(v) != 0 {
			converted := []cty.Value{}
			for _, value := range v {
				converted = append(converted, cty.StringVal(value))
			}
			return cty.ListVal(converted), true
		}
	case map[string]string:
		if len(v) != 0 {
			converted := map[string]cty.Value{}
			for mk, mv := range v {
				converted[mk] = cty.StringVal(mv)
			}
			return cty.MapVal(converted), true
		}
	case map[string][]string:
		if len(v) != 0 {
			convertedMap := map[string]cty.Value{}
			for mk, mv := range v {
				convertedList, ok := convertValue(mv)
				if !ok {
					return cv, false
				}
				convertedMap[mk] = convertedList
			}
			return cty.MapVal(convertedMap), true
		}
	case map[string]interface{}:
		if len(v) != 0 {
			convertedMap := map[string]cty.Value{}
			for mk, mv := range v {
				converted, ok := convertValue(mv)
				if !ok {
					return cv, false
				}
				convertedMap[mk] = converted
			}
			return cty.ObjectVal(convertedMap), true
		}
	default:
		panic(fmt.Sprintf("Unknown type for: %#t %#v", value, value))
	}

	return
}
