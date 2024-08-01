package update

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"path/filepath"
	"time"
)

// Automatic updates for hotify
const REPO = "s1adem4n/hotify"

type Updater struct {
	// Interval between update checks (API calls)
	CheckInterval time.Duration
	// Location of the local repository
	RepoPath string
	// Binary location, relative to the repository
	BinaryPath string
	// Branch to check for updates
	Branch string
	// Command to run after updating (eg. restart)
	UpdateCommand string
	// Current commit hash
	CommitHash string
}

type Commit struct {
	Hash string `json:"sha"`
}

func (u *Updater) CheckForUpdates() (bool, error) {
	resp, err := http.Get(
		fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", REPO, u.Branch),
	)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var commit Commit
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return false, err
	}

	return commit.Hash != u.CommitHash, nil
}

func (u *Updater) PullUpdates() error {
	cmd, err := exec.Command("git", "-C", u.RepoPath, "pull").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull updates: %s", string(cmd))
	}

	return nil
}

func (u *Updater) Update() error {
	if err := u.PullUpdates(); err != nil {
		return err
	}

	// build the binary
	cmd := exec.Command("./build.sh", u.BinaryPath)
	cmd.Dir = filepath.Join(u.RepoPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		logs := out.String()
		return fmt.Errorf("failed to build binary: %s, %s", err, logs)
	}

	if u.UpdateCommand != "" {
		cmd := exec.Command("bash", "-c", u.UpdateCommand)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run update command: %s", err)
		}
	}

	return nil
}

func (u *Updater) Run() {
	for {
		slog.Info("Checking for updates")

		needsUpdate, err := u.CheckForUpdates()
		if err != nil {
			slog.Error("Failed to check for updates", "error", err)
		}

		if needsUpdate {
			slog.Info("Update available, updating...")
			if err := u.Update(); err != nil {
				slog.Error("Failed to update", "error", err)
			}
		} else {
			slog.Info("No updates available")
		}

		time.Sleep(u.CheckInterval)
	}
}

func NewSystemdUpdater(hash string) *Updater {
	return &Updater{
		CheckInterval: time.Minute,
		RepoPath:      "src",
		BinaryPath:    "../hotify",
		Branch:        "main",
		CommitHash:    hash,
		UpdateCommand: "sudo systemctl restart hotify",
	}
}
