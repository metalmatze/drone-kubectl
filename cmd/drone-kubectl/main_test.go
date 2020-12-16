package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKubectlArgs(t *testing.T) {
	args := kubectlArgs("get pods")
	assert.Equal(t, []string{"get", "pods"}, args)
}

func TestKubectlFiles(t *testing.T) {
	args := kubectlArgs("apply", kubectlFiles([]string{}))
	assert.Equal(t, []string{"apply"}, args)
	args = kubectlArgs("apply", kubectlFiles([]string{"foo.yaml"}))
	assert.Equal(t, []string{"apply", "-f", "foo.yaml"}, args)
	args = kubectlArgs("apply", kubectlFiles([]string{"foo.yaml", "bar.yaml"}))
	assert.Equal(t, []string{"apply", "-f", "foo.yaml", "-f", "bar.yaml"}, args)

	// Don't overwrite files if already set
	args = kubectlArgs("apply -f app.yaml", kubectlFiles([]string{"foo.yaml"}))
	assert.Equal(t, []string{"apply", "-f", "app.yaml"}, args)
	args = kubectlArgs("apply --filename app.yaml", kubectlFiles([]string{"foo.yaml"}))
	assert.Equal(t, []string{"apply", "--filename", "app.yaml"}, args)
}

func TestKubectlNamespace(t *testing.T) {
	args := kubectlArgs("get pods", kubectlNamespace(""))
	assert.Equal(t, []string{"get", "pods"}, args)
	args = kubectlArgs("get pods", kubectlNamespace("app"))
	assert.Equal(t, []string{"get", "pods", "--namespace", "app"}, args)

	// Don't overwrite namespace if already set
	args = kubectlArgs("get pods -n drone", kubectlNamespace("app"))
	assert.Equal(t, []string{"get", "pods", "-n", "drone"}, args)
	args = kubectlArgs("get pods --namespace drone", kubectlNamespace("app"))
	assert.Equal(t, []string{"get", "pods", "--namespace", "drone"}, args)
}

func TestKubectlCommandTemplating(t *testing.T) {
	kubectl, err := kubectlCommand("get nodes")
	assert.NoError(t, err)
	assert.Equal(t, "get nodes", kubectl)

	os.Setenv("DRONE_COMMIT", "v1.2.3")

	kubectl, err = kubectlCommand("set image deployment/foo container=bar/baz:{{ .DroneCommit }}")
	assert.NoError(t, err)
	assert.Equal(t, "set image deployment/foo container=bar/baz:v1.2.3", kubectl)
}
