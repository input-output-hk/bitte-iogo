package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/jdxcode/netrc"
)

type LoginCmd struct {
	Cluster     string `arg:"--cluster,env:BITTE_CLUSTER,required"`
	cacheDir    string
	githubToken string
	role        string
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

	if admin {
		args.role = "admin"
	} else {
		args.role = "developer"
	}

	if err := loginNomad(args); err != nil {
		return err
	}

	if err := loginConsul(args); err != nil {
		return err
	}

	fmt.Printf(strings.TrimSpace(`
# Use this in your .envrc:
#
# eval "$(iogo login)"

export VAULT_TOKEN="%s"
export NOMAD_TOKEN="%s"
export CONSUL_HTTP_TOKEN="%s"
`), os.Getenv("VAULT_TOKEN"),
		os.Getenv("NOMAD_TOKEN"),
		os.Getenv("CONSUL_HTTP_TOKEN"))

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
		os.WriteFile(tokenPath, token, 0600)
	}()

	cmd.Run()

	_, err = exec.Command("vault", "token", "lookup").CombinedOutput()
	return err
}

func loginConsul(args *LoginCmd) error {
	tokenPath := filepath.Join(args.cacheDir, "consul.token")
	cachedContent, err := os.ReadFile(tokenPath)
	if err == nil {
		if err = os.Setenv("CONSUL_HTTP_TOKEN", string(cachedContent)); err != nil {
			return err
		}
	}

	_, err = exec.Command("consul", "acl", "token", "read", "-self").CombinedOutput()
	if err == nil {
		return nil
	}

	logger.Println("Obtaining and caching Consul token in " + tokenPath)

	output, err := exec.Command("vault", "read", "-field", "token", "consul/creds/"+args.role).CombinedOutput()
	if err != nil {
		return err
	}

	os.WriteFile(tokenPath, output, 0600)

	return os.Setenv("CONSUL_HTTP_TOKEN", string(output))
}

func loginNomad(args *LoginCmd) error {
	tokenPath := filepath.Join(args.cacheDir, "nomad.token")
	cachedContent, err := os.ReadFile(tokenPath)
	if err == nil {
		if err = os.Setenv("NOMAD_TOKEN", string(cachedContent)); err != nil {
			return err
		}
	}

	_, err = exec.Command("nomad", "acl", "token", "self").CombinedOutput()
	if err == nil {
		return nil
	}

	logger.Println("Obtaining and caching Nomad token")

	output, err := exec.Command("vault", "read", "-field", "secret_id", "nomad/creds/"+args.role).CombinedOutput()
	if err != nil {
		return err
	}

	os.WriteFile(tokenPath, output, 0600)

	return os.Setenv("NOMAD_TOKEN", string(output))
}

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
	output, err := exec.Command("vault", "token", "lookup", "-format", "json").CombinedOutput()
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
