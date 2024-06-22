package git

import "os/exec"

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
