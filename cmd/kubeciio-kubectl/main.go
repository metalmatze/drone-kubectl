package main

import (
	"bytes"
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
	"time"
	"unicode/utf8"

	"github.com/joho/godotenv"
	"github.com/pinzolo/casee"
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
	debugFlag       = "debug"
	debugEnvVar     = "PLUGIN_DEBUG,DEBUG"
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
	app.Version = "0.2.0"

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
		cli.BoolFlag{
			Name:   debugFlag,
			EnvVar: debugEnvVar,
			Usage:  "print out some sensitive debug info",
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	kubectl, err := kubectlCommand(c.String(kubectlFlag))
	if err != nil {
		return err
	}

	kubeconfig := ""

	kube64 := os.Getenv("KUBECONFIG")
	isInClusterConfig := kube64 == ""

	if isInClusterConfig {
		log.Println("Using in-cluster credentials")
	} else {
		log.Println("Decoding kubeconfig from secret")

		kube, err := base64.StdEncoding.DecodeString(kube64)
		if err != nil {
			return errors.Wrap(err, "failed to base64 decode kubeconfig from envvar")
		}

		if c.Bool(debugFlag) {
			log.Printf("decoded KUBECONFIG:\n%s", kube)
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
		if !isInClusterConfig {
			cmd.Env = []string{"KUBECONFIG=" + kubeconfig}
		}
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

// kubectlCommand makes sure there's not an empty string and executes templating if necessary
func kubectlCommand(command string) (string, error) {
	if command == "" {
		return "", errors.New("no kubectl command specific")
	}

	// If kubectl command contains {{ }} then enable templating on the kubectl command
	if strings.Contains(command, "{{") && strings.Contains(command, "}}") {
		tmpl, err := generateTemplate(command)
		if err != nil {
			return "", errors.Wrap(err, "failed to generate template from kubectl command")
		}
		return tmpl, err
	}
	return command, nil
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
				path, err := generateTemplateFile(t)
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

func generateTemplateFile(path string) (string, error) {
	tc, err := ioutil.ReadFile(path)
	if err != nil {
		return "", errors.Wrap(err, "failed to read file")
	}

	tmpl, err := generateTemplate(string(tc))
	if err != nil {
		return "", errors.Wrap(err, "failed to generate template from file")
	}

	tmpfile, err := ioutil.TempFile("", filepath.Base(path))
	if err != nil {
		return "", errors.Wrap(err, "failed to create tmp file for template")
	}

	if _, err := tmpfile.WriteString(tmpl); err != nil {
		return "", errors.Wrap(err, "failed to write template to tmp file")
	}

	if err := tmpfile.Close(); err != nil {
		return "", errors.Wrap(err, "failed to close tmp file for template")
	}

	return tmpfile.Name(), nil
}

func generateTemplate(t string) (string, error) {
	tmpl := template.New("template").Funcs(map[string]interface{}{
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"replace":    strings.Replace,
		"split":      strings.Split,
		"trim":       strings.Trim,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,
		"toTitle":    strings.ToTitle,
		"datetime":   templateDatetime,
		"trunc":      templateTruncate,
		"b64enc":     templateB64enc,
		"b64dec":     templateB64dec,
	})

	tmpl, err := tmpl.Parse(t)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse template")
	}

	w := bytes.NewBuffer(nil)
	if err := tmpl.Execute(w, environmentVariables()); err != nil {
		return "", errors.Wrap(err, "failed to execute template")
	}

	return w.String(), err
}

func environmentVariables() map[string]string {
	variables := make(map[string]string, len(os.Environ()))

	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		variables[casee.ToPascalCase(pair[0])] = pair[1]
	}

	return variables
}

func templateDatetime(timestamp float64, layout, zone string) string {
	if zone == "" {
		return time.Unix(int64(timestamp), 0).Format(layout)
	}

	loc, err := time.LoadLocation(zone)

	if err != nil {
		return time.Unix(int64(timestamp), 0).Local().Format(layout)
	}

	return time.Unix(int64(timestamp), 0).In(loc).Format(layout)
}

func templateTruncate(s string, len int) string {
	if utf8.RuneCountInString(s) <= len {
		return s
	}

	runes := []rune(s)

	return string(runes[:len])
}

func templateB64enc(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func templateB64dec(s string) string {
	data, err := base64.StdEncoding.DecodeString(s)

	if err != nil {
		return s
	}

	return string(data)
}
