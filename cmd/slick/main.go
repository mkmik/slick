package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	kongcompletion "github.com/jotaen/kong-completion"

	"mkm.pub/slick/internal/markdown"
	slackclient "mkm.pub/slick/internal/slack"
)

type CLI struct {
	Token      string                   `help:"Slack API token." env:"SLICK_TOKEN" required:""`
	Cat        CatCmd                   `cmd:"" help:"Fetch and display a Slack thread as markdown."`
	Completion kongcompletion.Completion `cmd:"" help:"Output shell completion code."`
}

type CatCmd struct {
	URL string `arg:"" help:"Slack thread URL." required:""`
}

func (c *CatCmd) Run(globals *CLI) error {
	client := slackclient.New(globals.Token)
	thread, err := client.FetchThread(c.URL)
	if err != nil {
		return err
	}
	fmt.Print(markdown.Render(thread))
	return nil
}

func main() {
	var cli CLI
	app := kong.Must(&cli,
		kong.Name("slick"),
		kong.Description("Fetch and display Slack threads as markdown for LLM agents."),
		kong.UsageOnError(),
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
