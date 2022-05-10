package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jdxcode/netrc"
)

type LoginCmd struct {
	Cluster     string `arg:"--cluster,env:BITTE_CLUSTER,required"`
	cacheDir    string
	githubToken string
	role        string
}

const githubApi = "api.github.com"

func (l *LoginCmd) runLogin() error {
	dir := cacheDir(l.Cluster)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	l.cacheDir = dir
	if err := l.setGithubToken(); err != nil {
		return err
	}

	if err := l.loginVault(); err != nil {
		return err
	}

	admin, err := isAdmin()
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}

	if admin {
		l.role = "admin"
		if strings.EqualFold(os.Getenv("BITTE_PROVIDER"), "AWS") {
			os.Unsetenv("AWS_PROFILE")
			if err = os.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/dev/null"); err != nil {
				return err
			}
			wg.Add(1)
			go l.loginAWS(wg)
		}
	} else {
		l.role = "developer"
	}

	wg.Add(2)
	go l.loginNomad(wg)
	go l.loginConsul(wg)
	wg.Wait()

	fmt.Printf(strings.TrimSpace(`
# Use this in your .envrc:
#
# eval "$(iogo login)"

export GITHUB_TOKEN="%s"
export VAULT_TOKEN="%s"
export NOMAD_TOKEN="%s"
export CONSUL_HTTP_TOKEN="%s"
export AWS_ACCESS_KEY_ID="%s"
export AWS_SECRET_ACCESS_KEY="%s"
`), os.Getenv("GITHUB_TOKEN"),
		os.Getenv("VAULT_TOKEN"),
		os.Getenv("NOMAD_TOKEN"),
		os.Getenv("CONSUL_HTTP_TOKEN"),
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"))

	return nil
}

func (l *LoginCmd) setGithubToken() error {
	usr, err := user.Current()
	if err != nil {
		return err
	}

	if rc, err := netrc.Parse(filepath.Join(usr.HomeDir, ".netrc")); err != nil {
		return err
	} else if machine := rc.Machine(githubApi); machine != nil {
		if password := machine.Get("password"); password != "" {
			if err = os.Setenv("GITHUB_TOKEN", password); err != nil {
				return err
			}

			l.githubToken = password

			return nil
		} else {
			return fmt.Errorf("No password for %s found in ~/.netrc", githubApi)
		}
	}

	return fmt.Errorf("No entry for %s found in ~/.netrc", githubApi)
}

func (l *LoginCmd) loginVault() error {
	tokenPath := filepath.Join(l.cacheDir, "vault.token")
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
		"token="+l.githubToken)

	stdout := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logger.Println(stdout.String())
		return err
	}

	token := stdout.String()
	if err := os.Setenv("VAULT_TOKEN", token); err != nil {
		return err
	}

	if err := os.WriteFile(tokenPath, []byte(token), 0600); err != nil {
		return err
	}

	out, err := exec.Command("vault", "token", "lookup").CombinedOutput()
	if err != nil {
		logger.Println(string(out))
		return err
	}

	return nil
}

func (l *LoginCmd) loginAWS(wg *sync.WaitGroup) {
	defer wg.Done()
	if err := l.loginAWSInner(); err != nil {
		logger.Println("Failed logging into AWS:", err.Error())
	}
}

func (l *LoginCmd) loginAWSInner() error {
	keyPath := filepath.Join(l.cacheDir, "aws.key")
	secretPath := filepath.Join(l.cacheDir, "aws.secret")

	if key, err := os.ReadFile(keyPath); err != nil {
	} else if err := os.Setenv("AWS_ACCESS_KEY_ID", string(key)); err != nil {
		return err
	}

	if secret, err := os.ReadFile(secretPath); err != nil {
	} else if err := os.Setenv("AWS_SECRET_ACCESS_KEY", string(secret)); err != nil {
		return err
	}

	_, err := exec.Command("aws", "iam", "get-user").CombinedOutput()
	if err == nil {
		return nil
	}

	logger.Println("Obtaining and caching AWS keys")

	output, err := exec.Command("vault", "read", "aws/creds/admin", "-format=json").CombinedOutput()
	if err != nil {
		logger.Println("failed `vault read aws/creds/admin -format=json`:", string(output))
		return err
	}

	Keys := &AWSKeys{}

	if err := json.Unmarshal(output, Keys); err != nil {
		return err
	}

	key := Keys.Data.Access_Key
	if err := os.Setenv("AWS_ACCESS_KEY_ID", string(key)); err != nil {
		return err
	}
	if err := os.WriteFile(keyPath, []byte(key), 0600); err != nil {
		return err
	}

	secret := Keys.Data.Secret_Key
	if err := os.Setenv("AWS_SECRET_ACCESS_KEY", string(secret)); err != nil {
		return err
	}
	if err := os.WriteFile(secretPath, []byte(secret), 0600); err != nil {
		return err
	}

	return nil
}

func (l *LoginCmd) loginConsul(wg *sync.WaitGroup) {
	defer wg.Done()
	if err := l.loginConsulInner(); err != nil {
		logger.Println("Failed logging into Consul:", err.Error())
	}
}

func (l *LoginCmd) loginConsulInner() error {
	tokenPath := filepath.Join(l.cacheDir, "consul.token")
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

	output, err := exec.Command("vault", "read", "-field", "token", "consul/creds/"+l.role).CombinedOutput()
	if err != nil {
		return err
	}

	if err := os.WriteFile(tokenPath, output, 0600); err != nil {
		return err
	}

	return os.Setenv("CONSUL_HTTP_TOKEN", string(output))
}

func (l *LoginCmd) loginNomad(wg *sync.WaitGroup) {
	defer wg.Done()

	if err := l.loginNomadInner(); err != nil {
		logger.Println("Failed logging into Nomad:", err.Error())
	}
}

func (l *LoginCmd) loginNomadInner() error {
	tokenPath := filepath.Join(l.cacheDir, "nomad.token")
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

	output, err := exec.Command("vault", "read", "-field", "secret_id", "nomad/creds/"+l.role).CombinedOutput()
	if err != nil {
		logger.Println(string(output))
		return err
	}

	if err := os.WriteFile(tokenPath, output, 0600); err != nil {
		return err
	}

	if err := os.Setenv("NOMAD_TOKEN", string(output)); err != nil {
		return err
	}

	return nil
}

func cacheDir(cluster string) string {
	root := os.Getenv("XDG_CACHE_HOME")
	if root == "" {
		root = filepath.Join(os.Getenv("HOME"), ".cache")
	}

	return filepath.Join(root, "bitte", cluster, "tokens")
}

type VaultToken struct {
	Data VaultTokenData
}

type VaultTokenData struct {
	Policies []string
}

type AWSKeys struct {
	Data AWSKeysData
}

type AWSKeysData struct {
	Access_Key string
	Secret_Key string
}

func isAdmin() (bool, error) {
	output, err := exec.Command("vault", "token", "lookup", "-format", "json").CombinedOutput()
	if err != nil {
		return false, err
	}

	vaultToken := &VaultToken{}
	if err := json.Unmarshal(output, vaultToken); err != nil {
		return false, err
	}

	for _, policy := range vaultToken.Data.Policies {
		if policy == "admin" {
			return true, nil
		}
	}

	return false, nil
}
