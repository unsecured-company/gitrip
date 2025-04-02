package main

import (
	"fmt"
	"os"
	"unsecured.company/gitrip/internal/application"
	"unsecured.company/gitrip/internal/git"
)

var buildVersion = application.Version // Other default values are set in config.go

type GitRip struct {
	app *application.App
}

func main() {
	_, _ = fmt.Fprintln(os.Stderr, "GitRip "+buildVersion+" | https://unsecured.company\n")
	var mErr *application.MultiErr

	gr := GitRip{}
	gr.app, mErr = application.NewApp(os.Args)

	if !mErr.HasErrors() && gr.app.Cfg.Command != application.CmdHelp {
		gr.app.Cfg.PrintConfigurationText(gr.app.Out)
		mErr.Add(gr.Run())
	}

	if mErr.HasErrors() {
		mErr.Dump(gr.app.Out)
		os.Exit(1)
	}
}

func (gr *GitRip) Run() (err error) {
	switch gr.app.Cfg.Command {
	case application.CmdCheck:
		err = gr.runCheck()
	case application.CmdFetch:
		err = gr.runFetch()
	case application.CmdIndex:
		err = git.RunIndexDump(gr.app)
	case application.CmdHelp, "":
		os.Exit(0)
	default:
		err = fmt.Errorf("unknown command '%s'", gr.app.Cfg.Command)
	}

	return
}

func (gr *GitRip) runCheck() (err error) {
	run := git.NewChecker(gr.app)
	err = run.Run()

	return
}

func (gr *GitRip) runFetch() (err error) {
	run := git.NewDumper(gr.app)
	err = run.Run()

	return
}
