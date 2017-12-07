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

	data := struct {
		DroneArch               string
		DroneBranch             string
		DroneBuildCreated       string
		DroneBuildEvent         string
		DroneBuildFinished      string
		DroneBuildLink          string
		DroneBuildNumber        string
		DroneBuildStarted       string
		DroneBuildStatus        string
		DroneCommit             string
		DroneCommitAuthor       string
		DroneCommitAuthorAvatar string
		DroneCommitAuthorEmail  string
		DroneCommitBranch       string
		DroneCommitLink         string
		DroneCommitMessage      string
		DroneCommitRef          string
		DroneCommitSha          string
		DroneJobFinished        string
		DroneJobNumber          string
		DroneJobStarted         string
		DroneJobStatus          string
		DroneMachine            string
		DroneNetrcMachine       string
		DroneNetrcPassword      string
		DroneNetrcUsername      string
		DroneParentBuildNumber  string
		DronePrevBuildNumber    string
		DronePrevBuildStatus    string
		DronePrevCommitSha      string
		DroneRemoteURL          string
		DroneRepo               string
		DroneRepoLink           string
		DroneRepoName           string
		DroneRepoOwner          string
		DroneRepoPrivate        string
		DroneRepoScm            string
		DroneWorkspace          string
	}{
		DroneArch:               os.Getenv("DRONE_ARCH"),
		DroneBranch:             os.Getenv("DRONE_BRANCH"),
		DroneBuildCreated:       os.Getenv("DRONE_BUILD_CREATED"),
		DroneBuildEvent:         os.Getenv("DRONE_BUILD_EVENT"),
		DroneBuildFinished:      os.Getenv("DRONE_BUILD_FINISHED"),
		DroneBuildLink:          os.Getenv("DRONE_BUILD_LINK"),
		DroneBuildNumber:        os.Getenv("DRONE_BUILD_NUMBER"),
		DroneBuildStarted:       os.Getenv("DRONE_BUILD_STARTED"),
		DroneBuildStatus:        os.Getenv("DRONE_BUILD_STATUS"),
		DroneCommit:             os.Getenv("DRONE_COMMIT"),
		DroneCommitAuthor:       os.Getenv("DRONE_COMMIT_AUTHOR"),
		DroneCommitAuthorAvatar: os.Getenv("DRONE_COMMIT_AUTHOR_AVATAR"),
		DroneCommitAuthorEmail:  os.Getenv("DRONE_COMMIT_AUTHOR_EMAIL"),
		DroneCommitBranch:       os.Getenv("DRONE_COMMIT_BRANCH"),
		DroneCommitLink:         os.Getenv("DRONE_COMMIT_LINK"),
		DroneCommitMessage:      os.Getenv("DRONE_COMMIT_MESSAGE"),
		DroneCommitRef:          os.Getenv("DRONE_COMMIT_REF"),
		DroneCommitSha:          os.Getenv("DRONE_COMMIT_SHA"),
		DroneJobFinished:        os.Getenv("DRONE_JOB_FINISHED"),
		DroneJobNumber:          os.Getenv("DRONE_JOB_NUMBER"),
		DroneJobStarted:         os.Getenv("DRONE_JOB_STARTED"),
		DroneJobStatus:          os.Getenv("DRONE_JOB_STATUS"),
		DroneMachine:            os.Getenv("DRONE_MACHINE"),
		DroneNetrcMachine:       os.Getenv("DRONE_NETRC_MACHINE"),
		DroneNetrcPassword:      os.Getenv("DRONE_NETRC_PASSWORD"),
		DroneNetrcUsername:      os.Getenv("DRONE_NETRC_USERNAME"),
		DroneParentBuildNumber:  os.Getenv("DRONE_PARENT_BUILD_NUMBER"),
		DronePrevBuildNumber:    os.Getenv("DRONE_PREV_BUILD_NUMBER"),
		DronePrevBuildStatus:    os.Getenv("DRONE_PREV_BUILD_STATUS"),
		DronePrevCommitSha:      os.Getenv("DRONE_PREV_COMMIT_SHA"),
		DroneRemoteURL:          os.Getenv("DRONE_REMOTE_URL"),
		DroneRepo:               os.Getenv("DRONE_REPO"),
		DroneRepoLink:           os.Getenv("DRONE_REPO_LINK"),
		DroneRepoName:           os.Getenv("DRONE_REPO_NAME"),
		DroneRepoOwner:          os.Getenv("DRONE_REPO_OWNER"),
		DroneRepoPrivate:        os.Getenv("DRONE_REPO_PRIVATE"),
		DroneRepoScm:            os.Getenv("DRONE_REPO_SCM"),
		DroneWorkspace:          os.Getenv("DRONE_WORKSPACE"),
	}

	tmpfile, err := ioutil.TempFile("", filepath.Base(path))
	if err != nil {
		return "", errors.Wrap(err, "failed to create tmp file for template")
	}

	if err := tmpl.Execute(tmpfile, data); err != nil {
		return "", errors.Wrap(err, "failed to generate file from template")
	}

	if err := tmpfile.Close(); err != nil {
		return "", errors.Wrap(err, "failed to close tmp file for template")
	}

	return tmpfile.Name(), nil
}
