package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/jdxcode/netrc"
)

type LoginCmd struct {
	Cluster     string `arg:"--cluster,env:BITTE_CLUSTER,required"`
	cacheDir    string
	githubToken string
}

const githubApi = "api.github.com"

func runLogin(args *LoginCmd) error {
	ght, err := githubToken()
	if err != nil {
		return err
	}
	args.githubToken = ght

	dir := cacheDir(args.Cluster)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	args.cacheDir = dir

	if err := loginVault(args); err != nil {
		return err
	}

	admin, err := isAdmin()
	if err != nil {
		return err
	}

	fmt.Println("isAdmin", admin)

	if err := loginNomad(args); err != nil {
		return err
	}

	if err := loginConsul(args); err != nil {
		return err
	}

	return nil
}

func loginVault(args *LoginCmd) error {
	tokenPath := filepath.Join(args.cacheDir, "vault.token")
	content, err := os.ReadFile(tokenPath)
	if err == nil {
		if err = os.Setenv("VAULT_TOKEN", string(content)); err != nil {
			return err
		}
	}

	logger.Println("vault token lookup")
	_, err = exec.Command("vault", "token", "lookup").CombinedOutput()
	if err == nil {
		return nil
	}

	logger.Println("Obtaining and caching Vault token")

	cmd := exec.Command(
		"vault", "login",
		"-no-store",
		"-token-only",
		"-method=github",
		"-path=github-employees",
		"token=-")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		stdin.Write([]byte(args.githubToken))
		stdin.Close()
	}()

	go func() {
		token, _ := io.ReadAll(stdout)
		os.Setenv("VAULT_TOKEN", string(token))
	}()

	cmd.Run()

	logger.Println("vault token lookup")
	_, err = exec.Command("vault", "token", "lookup").CombinedOutput()
	if err == nil {
		return nil
	}

	return nil
}

func loginNomad(args *LoginCmd) error  { return nil }
func loginConsul(args *LoginCmd) error { return nil }

func cacheDir(cluster string) string {
	root := os.Getenv("XDG_CACHE_HOME")
	if root == "" {
		root = ".direnv"
	}

	return filepath.Join(root, "bitte", cluster, "tokens")
}

type VaultToken struct {
	Data VaultTokenData
}

type VaultTokenData struct {
	Policies []string
}

func isAdmin() (bool, error) {
	cmd := exec.Command("vault", "token", "lookup", "-format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}

	vaultToken := &VaultToken{}
	json.Unmarshal(output, vaultToken)

	for _, policy := range vaultToken.Data.Policies {
		if policy == "admin" {
			return true, nil
		}
	}

	return false, nil
}

func githubToken() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	rc, err := netrc.Parse(filepath.Join(usr.HomeDir, ".netrc"))

	if machine := rc.Machine(githubApi); machine != nil {
		if password := machine.Get("password"); password != "" {
			return password, nil
		} else {
			return "", fmt.Errorf("No password for %s found in ~/.netrc", githubApi)
		}
	}

	return "", fmt.Errorf("No entry for %s found in ~/.netrc", githubApi)
}
