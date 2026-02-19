package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/alecthomas/kong"
	kongcompletion "github.com/jotaen/kong-completion"

	"mkm.pub/slick/internal/auth"
	"mkm.pub/slick/internal/markdown"
	slackclient "mkm.pub/slick/internal/slack"
)

// set by goreleaser or equivalent tool
var version = "(devel)"

func getVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return version
}

type CLI struct {
	Token      string                   `help:"Slack API token (overrides stored token from 'login')." env:"SLICK_TOKEN"`
	Cat        CatCmd                   `cmd:"" help:"Fetch and display a Slack thread as markdown."`
	Login      LoginCmd                 `cmd:"" help:"Authenticate with Slack using OAuth."`
	Completion kongcompletion.Completion `cmd:"" help:"Output shell completion code."`
	Version    kong.VersionFlag         `name:"version" help:"Print version."`
}

// resolveToken returns the token from --token/SLICK_TOKEN, falling back to the stored token from 'slick login'.
func resolveToken(globals *CLI) (string, error) {
	if globals.Token != "" {
		return globals.Token, nil
	}
	token, err := auth.LoadToken()
	if err != nil {
		return "", fmt.Errorf("no token available: set SLICK_TOKEN or run 'slick login'")
	}
	return token, nil
}

type CatCmd struct {
	URL string `arg:"" help:"Slack thread URL." required:""`
}

func (c *CatCmd) Run(globals *CLI) error {
	token, err := resolveToken(globals)
	if err != nil {
		return err
	}
	client := slackclient.New(token)
	thread, err := client.FetchThread(c.URL)
	if err != nil {
		return err
	}
	fmt.Print(markdown.Render(thread))
	return nil
}

type LoginCmd struct {
	ClientID     string `help:"Slack app client ID." env:"SLICK_CLIENT_ID" required:""`
	ClientSecret string `help:"Slack app client secret." env:"SLICK_CLIENT_SECRET" required:""`
}

func (c *LoginCmd) Run(globals *CLI) error {
	token, err := auth.Login(c.ClientID, c.ClientSecret)
	if err != nil {
		return err
	}
	if err := auth.SaveToken(token); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}
	path, _ := auth.TokenPath()
	fmt.Fprintf(os.Stderr, "Token saved to %s\n", path)
	return nil
}

func main() {
	var cli CLI
	app := kong.Must(&cli,
		kong.Name("slick"),
		kong.Description("Fetch and display Slack threads as markdown for LLM agents."),
		kong.UsageOnError(),
		kong.Vars{"version": getVersion()},
	)
	kongcompletion.Register(app)
	ctx, err := app.Parse(os.Args[1:])
	if err != nil {
		app.FatalIfErrorf(err)
	}
	if err := ctx.Run(&cli); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
