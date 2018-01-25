package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	binary = "kubectl"

	dryRunFlag      = "dry-run"
	dryRunEnvVar    = "PLUGIN_DRY_RUN,DRY_RUN"
	filesFlag       = "files"
	filesEnvVar     = "PLUGIN_FILES,FILES"
	kubectlFlag     = "kubectl"
	kubectlEnvVar   = "PLUGIN_KUBECTL,KUBECTL"
	namespaceFlag   = "namespace"
	namespaceEnvVar = "PLUGIN_NAMESPACE,NAMESPACE"
	templatesFlag   = "templates"
	templatesEnvVar = "PLUGIN_TEMPLATES,TEMPLATES"
)

func main() {
	// Load env-file if it exists first
	if env := os.Getenv("PLUGIN_ENV_FILE"); env != "" {
		godotenv.Load(env)
	}

	app := cli.NewApp()
	app.Name = "KubeCI kubectl"
	app.Usage = "Run kubectl in your pipeline"
	app.Action = run
	app.Version = "0.0.0"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   dryRunFlag,
			EnvVar: dryRunEnvVar,
			Usage:  "don't actually call kubectl",
		},
		cli.StringSliceFlag{
			Name:   filesFlag,
			EnvVar: filesEnvVar,
			Usage:  "the files to use with kubectl",
		},
		cli.StringFlag{
			Name:   kubectlFlag,
			EnvVar: kubectlEnvVar,
			Usage:  "the kubectl command to execute",
		},
		cli.StringFlag{
			Name:   namespaceFlag,
			EnvVar: namespaceEnvVar,
			Usage:  "the namespace used by kubectl",
		},
		cli.StringSliceFlag{
			Name:   templatesFlag,
			EnvVar: templatesEnvVar,
			Usage:  "the template files to use with kubectl",
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	kubectl := c.String(kubectlFlag)
	if kubectl == "" {
		return errors.New("no kubectl command specific")
	}

	kubeconfig := ""

	kube64 := os.Getenv("KUBECONFIG")
	if kube64 == "" {
		fmt.Println("Using in-cluster credentials")
	} else {
		fmt.Println("Decoding kubeconfig from secret")

		kube, err := base64.StdEncoding.DecodeString(kube64)
		if err != nil {
			return errors.Wrap(err, "failed to base64 decode kubeconfig from envvar")
		}

		tmpfile, err := ioutil.TempFile("", "kubeconfig")
		if err != nil {
			return errors.Wrap(err, "failed to create tmp file for kubeconfig")
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write(kube); err != nil {
			return errors.Wrap(err, "failed to write to tmp kubeconfig file")
		}

		if err := tmpfile.Close(); err != nil {
			return errors.Wrap(err, "failed to close tmp kubeconfig file")
		}

		kubeconfig = tmpfile.Name()
	}

	// Unset KUBECONFIG if set to avoid leaking kubeconfig
	// We will only read it from file, ever
	os.Unsetenv("KUBECONFIG")

	args := kubectlArgs(kubectl,
		kubectlFiles(c.StringSlice(filesFlag)),
		kubectlNamespace(c.String(namespaceFlag)),
		kubectlTemplates(c.StringSlice(templatesFlag)),
	)

	if !c.Bool(dryRunFlag) {
		cmd := exec.CommandContext(context.TODO(), binary, args...)
		cmd.Env = []string{"KUBECONFIG=" + kubeconfig}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		fmt.Fprintf(os.Stdout, "+ %s\n", strings.Join(cmd.Args, " "))
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

type kubectlOption func([]string) []string

func kubectlArgs(kubectl string, options ...kubectlOption) []string {
	args := strings.Split(kubectl, " ")
	for _, opt := range options {
		args = opt(args)
	}
	return args
}

func kubectlFiles(files []string) kubectlOption {
	return func(args []string) []string {
		if !stringsContain(args, "--filename") && !stringsContain(args, "-f") {
			for _, f := range files {
				args = append(args, "-f", f)
			}
		}
		return args
	}
}

func kubectlNamespace(ns string) kubectlOption {
	return func(args []string) []string {
		if ns == "" {
			return args
		}
		if !stringsContain(args, "--namespace") && !stringsContain(args, "-n") {
			args = append(args, "--namespace", ns)
		}
		return args
	}
}

func kubectlTemplates(templates []string) kubectlOption {
	return func(args []string) []string {
		if !stringsContain(args, "--filename") && !stringsContain(args, "-f") {
			for _, t := range templates {
				path, err := generateTemplate(t)
				if err != nil {
					fmt.Println(t, err)
					continue
				}
				args = append(args, "-f", path)
			}
		}

		return args
	}
}

func stringsContain(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func generateTemplate(path string) (string, error) {
	tc, err := ioutil.ReadFile(path)
	if err != nil {
		return "", errors.Wrap(err, "failed to read file")
	}

	tmpl := template.New("template")
	tmpl.Funcs(map[string]interface{}{
		"truncate": func(len int, s string) string {
			return s[:len]
		},
	})

	tmpl, err = tmpl.Parse(string(tc))
	if err != nil {
		return "", errors.Wrap(err, "failed to parse template")
	}

	tmpfile, err := ioutil.TempFile("", filepath.Base(path))
	if err != nil {
		return "", errors.Wrap(err, "failed to create tmp file for template")
	}

	if err := tmpl.Execute(tmpfile, os.Environ()); err != nil {
		return "", errors.Wrap(err, "failed to generate file from template")
	}

	if err := tmpfile.Close(); err != nil {
		return "", errors.Wrap(err, "failed to close tmp file for template")
	}

	return tmpfile.Name(), nil
}
