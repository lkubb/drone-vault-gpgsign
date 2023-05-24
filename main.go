package main

import (
	"log"
	"os"

	"github.com/lkubb/drone-vault-gpgsign/plugin"

	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.App{
		Name:   "Vault GPG signing plugin for Drone CI",
		Action: run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "key",
				Usage:   "Named Vault GPG key to sign the artifacts with.",
				Value:   "drone",
				EnvVars: []string{"PLUGIN_KEY"},
			},
			&cli.BoolFlag{
				Name:    "armor",
				Usage:   "Write ASCII-armored detached signatures (.asc) instead of binary ones (.sig).",
				Value:   false,
				EnvVars: []string{"PLUGIN_ARMOR"},
			},
			&cli.StringFlag{
				Name:    "mount",
				Usage:   "Mount the GPG secret backend is mounted at.",
				Value:   "gpg",
				EnvVars: []string{"PLUGIN_MOUNT"},
			},
			&cli.StringFlag{
				Name:    "algo",
				Usage:   "Specifies the hash algorithm to use.",
				Value:   "sha2-256",
				EnvVars: []string{"PLUGIN_ALGO"},
			},
			&cli.StringFlag{
				Name:    "auth",
				Usage:   "Specifies the auth method to use with Vault. Valid: `token`, `approle`.",
				Value:   "token",
				EnvVars: []string{"PLUGIN_AUTH"},
			},
			&cli.StringFlag{
				Name:    "authmount",
				Usage:   "Specifies the mount the AppRole auth backend is mounted at.",
				Value:   "approle",
				EnvVars: []string{"PLUGIN_AUTHMOUNT"},
			},
			&cli.StringFlag{
				Name:    "roleid",
				Usage:   "Specifies the Vault AppRole RoleID to authenticate with.",
				EnvVars: []string{"PLUGIN_ROLEID"},
			},
			&cli.StringFlag{
				Name:    "secretid",
				Usage:   "Specifies the Vault AppRole SecretID to authenticate with.",
				EnvVars: []string{"PLUGIN_SECRETID"},
			},
			&cli.BoolFlag{
				Name:    "wrapped-secret",
				Usage:   "Indicates that the authentication secret is passed as a wrapping token.",
				Value:   false,
				EnvVars: []string{"PLUGIN_WRAPPED_SECRET"},
			},
			&cli.StringSliceFlag{
				Name:     "files",
				Usage:    "List of files to sign.",
				Required: true,
				EnvVars:  []string{"PLUGIN_FILES"},
			},
			&cli.StringSliceFlag{
				Name:    "exclude",
				Usage:   "List of exclude patterns.",
				EnvVars: []string{"PLUGIN_EXCLUDE"},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	conf := plugin.Config{
		Key:           c.String("key"),
		Armor:         c.Bool("armor"),
		Mount:         c.String("mount"),
		Algo:          c.String("algo"),
		Auth:          c.String("auth"),
		AuthMount:     c.String("authmount"),
		RoleID:        c.String("roleid"),
		SecretID:      c.String("secretid"),
		SecretWrapped: c.Bool("wrapped-secret"),
		Files:         c.StringSlice("files"),
		Exclude:       c.StringSlice("exclude"),
	}
	gpg, err := plugin.NewPlugin(conf)
	if err != nil {
		log.Fatal(err)
	}
	return gpg.Exec()
}
