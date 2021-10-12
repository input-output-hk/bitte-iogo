package main

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/nomad/api"
)

func cueExport() (*CueExport, error) {
	cueVet()

	cmd := exec.Command(cue, "export")
	output, err := cmd.CombinedOutput()

	if err != nil {
		if len(output) > 0 {
			fmt.Printf("%s\n", output)
		}

		panic(err)
	}

	export := &CueExport{}
	err = json.Unmarshal(output, export)

	return export, err
}

type JobWrapper struct {
	Job *api.Job
}

type CueExport struct {
	Rendered map[string]map[string]JobWrapper
}

func cue2hcl(namespace, job string) (*hclwrite.File, error) {
	cueVet()

	export, err := cueExport()
	if err != nil {
		return nil, err
	}

	if foundNamespace, ok := export.Rendered[namespace]; ok {
		if foundJob, ok := foundNamespace[job]; ok {
			return any2hcl("job", foundJob.Job)
		} else {
			return nil, fmt.Errorf("Missing job %s in namespace %s", job, namespace)
		}
	}

	return nil, fmt.Errorf("Missing namespace %s", namespace)
}

func cueVet() {
	cmd := exec.Command(cue, "vet", "-c", "./...")
	output, err := cmd.CombinedOutput()

	if len(output) > 0 {
		fmt.Printf("%s\n", output)
	}

	if err != nil {
		panic(err)
	}
}
