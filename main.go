package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/alexflint/go-arg"
)

var buildVersion = "dev"
var buildCommit = "dirty"

const cue = "cue"

var logger = log.New(os.Stderr, "DEBUG: ", log.LstdFlags)

type PlanCmd struct {
	Namespace string `arg:"--namespace,env:NOMAD_NAMESPACE,required"`
	Job       string `arg:"positional,env:NOMAD_JOB,required"`
	Output    string `arg:"-o" help:"output" placeholder:"FILE"`
}

type RenderCmd struct {
	Namespace string `arg:"--namespace,env:NOMAD_NAMESPACE,required"`
	Job       string `arg:"positional,env:NOMAD_JOB,required"`
	Output    string `arg:"-o" help:"output" placeholder:"FILE"`
}

type RunCmd struct {
	Namespace string `arg:"--namespace,env:NOMAD_NAMESPACE,required"`
	Job       string `arg:"positional,env:NOMAD_JOB,required"`
	Output    string `arg:"-o" help:"output" placeholder:"FILE"`
}

type ListJobsCmd struct {
	Output string `arg:"-o" help:"output" placeholder:"FILE"`
}

type ListNamespacesCmd struct {
	Output string `arg:"-o" help:"output" placeholder:"FILE"`
}

type iogo struct {
	Debug          bool               `arg:"--debug" help:"debugging output"`
	Plan           *PlanCmd           `arg:"subcommand:plan"`
	Render         *RenderCmd         `arg:"subcommand:render"`
	Run            *RunCmd            `arg:"subcommand:run"`
	ListJobs       *ListJobsCmd       `arg:"subcommand:list-jobs"`
	ListNamespaces *ListNamespacesCmd `arg:"subcommand:list-namespaces"`
	Login          *LoginCmd          `arg:"subcommand:login"`
	Json2Hcl       *Json2HclCmd       `arg:"subcommand:json2hcl"`
}

func Version() string {
	return fmt.Sprintf("%s (%s)", buildVersion, buildCommit)
}

func (iogo) Version() string {
	return fmt.Sprintf("iogo %s", Version())
}

func main() {
	args := &iogo{}
	parser, err := parseArgs(args)
	fail(parser, err)

	if args.Debug {
		logger.SetOutput(os.Stderr)
	}

	fail(parser, run(parser, args))
}

func fail(parser *arg.Parser, err error) {
	switch err {
	case nil:
		return
	case arg.ErrHelp:
		parser.WriteHelp(os.Stderr)
		os.Exit(0)
	case arg.ErrVersion:
		fmt.Fprintln(os.Stdout, Version())
		os.Exit(0)
	default:
		fmt.Fprint(os.Stderr, err, "\n")
		os.Exit(1)
	}
}

func parseArgs(args *iogo) (*arg.Parser, error) {
	parser, err := arg.NewParser(arg.Config{}, args)
	if err != nil {
		return nil, err
	}

	return parser, parser.Parse(os.Args[1:])
}

func run(parser *arg.Parser, args *iogo) error {
	switch {
	case args.Plan != nil:
		return runPlan(args.Plan)
	case args.Render != nil:
		return runRender(args.Render)
	case args.Run != nil:
		return runRun(args.Run)
	case args.ListJobs != nil:
		return runListJobs(args.ListJobs)
	case args.ListNamespaces != nil:
		return runListNamespaces(args.ListNamespaces)
	case args.Login != nil:
		return runLogin(args.Login)
	case args.Json2Hcl != nil:
		return runJson2Hcl(args.Json2Hcl)
	default:
		parser.WriteHelp(os.Stderr)
	}

	return nil
}

func runListJobs(args *ListJobsCmd) error {
	export, err := cueExport()
	if err != nil {
		return err
	}

	out, err := openOutput(args.Output)
	if err != nil {
		return err
	}

	for namespace, jobs := range export.Rendered {
		for job := range jobs {
			fmt.Fprintf(out, "%s %s\n", namespace, job)
		}
	}

	return nil
}

func runListNamespaces(args *ListNamespacesCmd) error {
	export, err := cueExport()
	if err != nil {
		return err
	}

	out, err := openOutput(args.Output)
	if err != nil {
		return err
	}

	for namespace := range export.Rendered {
		fmt.Fprintf(out, "%s\n", namespace)
	}

	return nil
}

func runRender(args *RenderCmd) error {
	export, err := cueExport()
	if err != nil {
		return err
	}

	if namespace, ok := export.Rendered[args.Namespace]; ok {
		if job, ok := namespace[args.Job]; ok {
			hcl, err := any2hcl("job", job.Job)
			if err != nil {
				return err
			}

			out, err := openOutput(args.Output)
			if err != nil {
				return err
			}

			_, err = hcl.WriteTo(out)
			return err
		} else {
			return errors.New("Missing job in namespace")
		}
	}

	return errors.New("Missing namespace")
}

func runRun(args *RunCmd) error {
	return nomadJobDo(args.Namespace, args.Job, args.Output, "run")
}

func runPlan(args *PlanCmd) error {
	return nomadJobDo(args.Namespace, args.Job, args.Output, "plan")
}

func nomadJobDo(namespace, job, output, action string) error {
	hcl, err := cue2hcl(namespace, job)
	if err != nil {
		return err
	}

	if isStdpipe(output) {
		cmd := exec.Command("nomad", "job", action, "-")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		stdin, err := cmd.StdinPipe()
		if err != nil {
			return err
		}

		go func() {
			_, err = hcl.WriteTo(stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error while while generating HCL:\n%s\n", err)
				os.Exit(1)
			}

			stdin.Close()
		}()

		return cmd.Run()
	} else {
		out, err := openOutput(output)
		if err != nil {
			return err
		}

		_, err = hcl.WriteTo(out)
		if err != nil {
			return err
		}

		cmd := exec.Command("nomad", "job", action, output)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
}

func openOutput(name string) (io.Writer, error) {
	if isStdpipe(name) {
		return os.Stdout, nil
	} else {
		return os.OpenFile(name, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	}
}

func openInput(name string) (io.Reader, error) {
	if isStdpipe(name) {
		return os.Stdin, nil
	} else {
		return os.Open(name)
	}
}

func isStdpipe(name string) bool {
	return name == "" || name == "-"
}
