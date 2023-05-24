package plugin

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/fatih/structs"
	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/approle"
	"github.com/mitchellh/mapstructure"
)

type Config struct {
	Key           string
	Armor         bool
	Mount         string
	Algo          string
	Auth          string
	AuthMount     string
	RoleID        string
	SecretID      string
	SecretWrapped bool
	Files         []string
	Exclude       []string
}

type Plugin struct {
	client *vault.Client
	Config Config
}

type SignRequest struct {
	Algorithm string `json:"algorithm" structs:"algorithm,omitempty"`
	Format    string `json:"format" structs:"format,omitempty"`
	Input     string `json:"input" structs:"input"`
}

type LogEntry struct {
	Address string `json:"address"`
	UUID    string `json:"uuid"`
}

type SignResponse struct {
	Signature string    `json:"signature"`
	LogEntry  *LogEntry `json:"log_entry"`
}

func NewPlugin(config Config) (*Plugin, error) {
	if v := os.Getenv("VAULT_ADDR"); v == "" {
		return nil, errors.New("Vault host was not specified. Set the `VAULT_ADDR` environment variable.")
	}
	if config.Auth == "approle" && config.RoleID == "" {
		return nil, errors.New("Configured auth method `approle` requires at least a role ID to be set.")
	}
	if v := os.Getenv("VAULT_TOKEN"); v == "" && config.Auth == "token" {
		return nil, errors.New("Auth method is `token` and Vault token was not specified. Set the `VAULT_TOKEN` environment variable or use AppRole auth.")
	}
	client, err := vault.NewClient(nil)
	if err != nil {
		return nil, fmt.Errorf("Failed instantiating Vault client: %w")
	}
	gpg := &Plugin{
		client: client,
		Config: config,
	}
	return gpg, nil
}

func (p *Plugin) Exec() error {
	files, err := findFiles(p.Config.Files, p.Config.Exclude)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		log.Println("No files to sign")
		return nil
	}
	for _, file := range files {
		log.Println("Signing file %s", file)
		err = p.sign(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Plugin) sign(file string) error {
	data, err := loadFileBase64(file)
	if err != nil {
		return err
	}
	resp, err := p.requestSignature(data)
	if err != nil {
		return err
	}
	err = p.writeSignature(file, resp.Signature)
	if err != nil {
		return err
	}
	return nil
}

func (p *Plugin) requestSignature(data string) (*SignResponse, error) {
	p.ensureAuth()
	endpoint := p.Config.Mount + "/sign/" + p.Config.Key
	req := SignRequest{
		Algorithm: p.Config.Algo,
		Input:     data,
	}
	if p.Config.Armor {
		req.Format = "ascii-armor"
	}
	payload := structs.New(req).Map()
	resp, err := p.client.Logical().Write(endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("Failed to request signature: %w", err)
	}
	if resp == nil {
		return nil, errors.New("Expected a response for signature request")
	}
	sigResp := &SignResponse{}
	err = mapstructure.Decode(resp.Data, sigResp)
	if err != nil {
		return nil, fmt.Errorf("Failed parsing the signature response: %w", err)
	}
	return sigResp, nil
}

func (p *Plugin) writeSignature(file, sig string) error {
	var sigBytes []byte
	if p.Config.Armor {
		file = file + ".asc"
		sigBytes = []byte(sig)
	} else {
		var err error
		file = file + ".sig"
		sigBytes, err = base64.StdEncoding.DecodeString(sig)
		if err != nil {
			return fmt.Errorf("Failed decoding base64-encoded signature: %w", err)
		}
	}
	err := os.WriteFile(file, sigBytes, 0o644)
	if err != nil {
		return fmt.Errorf("Failed writing signature: %w", err)
	}
	return nil
}

func (p *Plugin) ensureAuth() error {
	if p.client.Token() == "" {
		switch p.Config.Auth {
		case "token":
			return errors.New("Token authentication is requested, but no token was found")
		case "approle":
			err := p.approleLogin()
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("Invalid auth method `%s` configured", p.Config.Auth)
		}
		if p.client.Token() == "" {
			return errors.New("Authentication did not provide a token. This is most likely an internal error")
		}
	}
	return nil
}

func (p *Plugin) approleLogin() error {
	options := []auth.LoginOption{auth.WithMountPath(p.Config.AuthMount)}
	var secretID *auth.SecretID
	if p.Config.SecretID != "" {
		secretID.FromString = p.Config.SecretID
		if p.Config.SecretWrapped {
			options = append(options, auth.WithWrappingToken())
		}
	}
	appRoleAuth, err := auth.NewAppRoleAuth(
		p.Config.RoleID,
		secretID,
		options...,
	)
	if err != nil {
		return fmt.Errorf("Unable to initialize AppRole auth method: %w", err)
	}
	authInfo, err := p.client.Auth().Login(context.Background(), appRoleAuth)
	if err != nil {
		return fmt.Errorf("Failed login with AppRole auth method: %w", err)
	}
	if authInfo == nil {
		return fmt.Errorf("no auth info was returned after login")
	}
	if p.client.Token() != "" {
		return nil
	}
	return errors.New("AppRole login seemed to succeed, but the client still does not carry a token")
}
