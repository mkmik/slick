package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/alecthomas/kong"
	kongcompletion "github.com/jotaen/kong-completion"

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
	Token      string                   `help:"Slack API token." env:"SLICK_TOKEN" required:""`
	Cat        CatCmd                   `cmd:"" help:"Fetch and display a Slack thread as markdown."`
	Post       PostCmd                  `cmd:"" help:"Post a message to a Slack channel or thread."`
	Test       TestCmd                  `cmd:"" help:"Check that the token can perform an authenticated request."`
	Completion kongcompletion.Completion `cmd:"" help:"Output shell completion code."`
	Version    kong.VersionFlag         `name:"version" help:"Print version."`
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

type PostCmd struct {
	Target       string `arg:"" help:"Slack thread URL, channel ID, #channel-name, or @me for a DM to yourself." required:""`
	Message      string `short:"m" help:"Message text. Read from stdin when omitted."`
	Yes          bool   `short:"y" help:"Send without confirming. Without it, preview and ask on a terminal, or exit non-zero when there is nobody to ask."`
	NoDisclaimer bool   `help:"Omit the footer marking the message as sent by a tool. Already omitted for bot tokens."`
	Markdown     bool   `help:"Send the body as standard Markdown for Slack to render, enabling tables, headings and nested lists. Renders subtly differently from a normal message."`
}

// disclaimer is appended to posted messages so readers can tell a message was
// sent by a tool rather than typed in Slack.
const disclaimer = "_Sent using_ :magic:"

func (p *PostCmd) Run(globals *CLI) error {
	client := slackclient.New(globals.Token)
	target, err := slackclient.ParseTarget(p.Target)
	if err != nil {
		return err
	}
	// chat.postMessage opens a DM when the channel is a user ID, so posting to
	// your own ID is the way to send yourself a note.
	if target.ChannelID == "@me" {
		resp, err := client.AuthTest()
		if err != nil {
			return fmt.Errorf("resolving @me: %w", err)
		}
		target.ChannelID = resp.UserID
	}
	body, err := p.body()
	if err != nil {
		return err
	}
	// With --markdown, Slack does the rendering and the body is sent untouched.
	// The footer is appended after either path: it is already mrkdwn, and running
	// it through ToMrkdwn would only risk mangling it.
	text := body
	if !p.Markdown {
		text = markdown.ToMrkdwn(body)
		// Warn on stderr so it survives a preview but never pollutes the permalink
		// that callers capture from stdout.
		if lossy := markdown.Lossy(body); len(lossy) > 0 {
			fmt.Fprintf(os.Stderr, "%s: Slack mrkdwn cannot render %s; add --markdown\n", warningLabel(), strings.Join(lossy, ", "))
		}
	}
	// Bot messages are already visibly from an app, so the footer only earns its
	// place on user tokens, where the message is attributed to a person.
	if !p.NoDisclaimer && !slackclient.IsBotToken(globals.Token) {
		text += "\n\n" + disclaimer
	}

	where := target.ChannelID
	if target.ThreadTS != "" {
		where += " (thread " + target.ThreadTS + ")"
	}
	rendering := "mrkdwn"
	if p.Markdown {
		rendering = "markdown block"
	}
	if !p.Yes {
		fmt.Printf("would post to %s as %s:\n---\n%s\n---\n", where, rendering, text)
		ok, err := confirm()
		if err != nil {
			return fmt.Errorf("refusing to send without --yes")
		}
		if !ok {
			return fmt.Errorf("cancelled")
		}
	}

	link, err := client.Post(target, text, p.Markdown)
	if err != nil {
		return err
	}
	fmt.Println(link)
	return nil
}

// warningLabel returns "warning" in yellow, or plain when the escape codes would
// only end up as noise: stderr redirected to a file or pipe, or NO_COLOR set.
// See https://no-color.org.
func warningLabel() string {
	if os.Getenv("NO_COLOR") != "" {
		return "warning"
	}
	if st, err := os.Stderr.Stat(); err != nil || st.Mode()&os.ModeCharDevice == 0 {
		return "warning"
	}
	return "\033[33mwarning\033[0m"
}

// confirm asks whether to send, and reports an error when there is nobody to
// ask — leaving --yes as the only way through in a script or an agent.
//
// The answer comes from the controlling terminal rather than stdin, which has
// usually already been consumed by the message body.
func confirm() (bool, error) {
	if st, err := os.Stdout.Stat(); err != nil || st.Mode()&os.ModeCharDevice == 0 {
		return false, fmt.Errorf("stdout is not a terminal")
	}
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return false, err
	}
	defer tty.Close()

	fmt.Print("send? [y/N] ")
	// A read error leaves the answer empty, which reads as "no".
	line, _ := bufio.NewReader(tty).ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

// body resolves the message text from --message or stdin.
func (p *PostCmd) body() (string, error) {
	if p.Message != "" {
		return p.Message, nil
	}
	// Check for a terminal first, so an interactive run errors instead of
	// silently blocking on a stdin nobody is going to write to.
	if st, err := os.Stdin.Stat(); err == nil && st.Mode()&os.ModeCharDevice != 0 {
		return "", fmt.Errorf("no message: pipe it on stdin or pass -m")
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("reading message from stdin: %w", err)
	}
	text := strings.TrimSpace(string(b))
	if text == "" {
		return "", fmt.Errorf("no message: stdin was empty")
	}
	return text, nil
}

type TestCmd struct{}

func (t *TestCmd) Run(globals *CLI) error {
	client := slackclient.New(globals.Token)
	resp, err := client.AuthTest()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	fmt.Printf("OK: authenticated as %s (%s) (team: %s)\n", resp.User, resp.UserID, resp.Team)
	if scopes := client.Scopes(); scopes != "" {
		fmt.Printf("scopes: %s\n", strings.ReplaceAll(scopes, ",", ", "))
	} else {
		fmt.Println("scopes: none reported (session tokens are unscoped)")
	}
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
