package installer

import (
	"bytes"
	"fmt"
	"os/exec"
)

func checkKubectl() error {
	_, err := exec.LookPath("kubectl")
	if err != nil {
		return fmt.Errorf("kubectl not found in PATH — install it from https://kubernetes.io/docs/tasks/tools/")
	}
	return nil
}

func kubectl(context string, args ...string) (string, error) {
	cmdArgs := args
	if context != "" {
		cmdArgs = append([]string{"--context", context}, args...)
	}

	cmd := exec.Command("kubectl", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %s", err, stderr.String())
	}
	return stdout.String(), nil
}
