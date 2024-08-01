package git

import (
	"bytes"
	"fmt"
	"os/exec"
)

func CloneRepo(url string, dest string) error {
	cmd := exec.Command("git", "clone", url, dest)
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func PullRepo(dest string) error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = dest
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func IsNewestCommit(dest string) (bool, error) {
	cmd := exec.Command("git", "fetch", "origin", "main")
	cmd.Dir = dest
	err := cmd.Run()
	if err != nil {
		return false, err
	}

	var buf bytes.Buffer
	cmd = exec.Command("git", "diff", "--name-only", "HEAD", "FETCH_HEAD")
	cmd.Dir = dest
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err = cmd.Run()
	if err != nil {
		return false, fmt.Errorf("failed to check for changes: %s", buf.String())
	}

	if len(buf.Bytes()) > 0 {
		return false, nil
	}

	return true, nil
}
