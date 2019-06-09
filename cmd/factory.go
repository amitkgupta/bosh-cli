package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudfoundry/bosh-cli/completion"

	// Should only be imported here to avoid leaking use of goflags through project
	goflags "github.com/jessevdk/go-flags"
)

type Factory struct {
	deps  BasicDeps
	compl *completion.Completion
}

func NewFactory(deps BasicDeps) Factory {
	return Factory{deps: deps, compl: completion.NewCompletion()}
}

func (f Factory) New(args []string) (Cmd, error) {
	var cmdOpts interface{}

	boshOpts := &BoshOpts{}

	boshOpts.VersionOpt = func() error {
		return &goflags.Error{
			Type:    goflags.ErrHelp,
			Message: fmt.Sprintf("version %s\n", VersionLabel),
		}
	}

	parser := goflags.NewParser(boshOpts, goflags.HelpFlag|goflags.PassDoubleDash)

	// Calculate optimal padding between sections in help output.
	descPaddingLength := 0
	namePaddingLength := 0
	for _, c := range parser.Commands() {
		if descPaddingLength < len(c.ShortDescription) {
			descPaddingLength = len(c.ShortDescription)
		}

		if namePaddingLength < len(c.Name) {
			namePaddingLength = len(c.Name)
		}
	}

	for _, c := range parser.Commands() {
		// Construct tab completion text.
		pad := namePaddingLength - len(c.Name)
		cmd := c.Name + strings.Repeat(" ", pad) + " - " + c.ShortDescription

		f.compl.AddCommand(cmd)

		// Construct help subcommand text.
		pad = descPaddingLength - len(c.ShortDescription) + 1
		docsURL := "https://bosh.io/docs/cli-v2#" + c.Name
		c.LongDescription = c.ShortDescription + "\n\n" + docsURL
		c.ShortDescription += strings.Repeat(" ", pad) + docsURL
	}

	parser.CommandHandler = func(command goflags.Commander, extraArgs []string) error {
		if opts, ok := command.(*SSHOpts); ok {
			if len(opts.Command) == 0 {
				opts.Command = extraArgs
				extraArgs = []string{}
			}
		}

		if opts, ok := command.(*AliasEnvOpts); ok {
			opts.URL = boshOpts.EnvironmentOpt
			opts.CACert = boshOpts.CACertOpt
		}

		if opts, ok := command.(*EventsOpts); ok {
			opts.Deployment = boshOpts.DeploymentOpt
		}

		if opts, ok := command.(*VMsOpts); ok {
			opts.Deployment = boshOpts.DeploymentOpt
		}

		if opts, ok := command.(*InstancesOpts); ok {
			opts.Deployment = boshOpts.DeploymentOpt
		}

		if opts, ok := command.(*TasksOpts); ok {
			opts.Deployment = boshOpts.DeploymentOpt
		}

		if opts, ok := command.(*TaskOpts); ok {
			opts.Deployment = boshOpts.DeploymentOpt
		}

		if opts, ok := command.(*CancelTasksOpts); ok {
			opts.Deployment = boshOpts.DeploymentOpt
		}

		if len(extraArgs) > 0 {
			errMsg := "Command '%T' does not support extra arguments: %s"
			return fmt.Errorf(errMsg, command, strings.Join(extraArgs, ", "))
		}

		cmdOpts = command

		return nil
	}

	boshOpts.SSH.GatewayFlags.UUIDGen = f.deps.UUIDGen
	boshOpts.SCP.GatewayFlags.UUIDGen = f.deps.UUIDGen
	boshOpts.Logs.GatewayFlags.UUIDGen = f.deps.UUIDGen

	goflags.FactoryFunc = func(val interface{}) {
		stype := reflect.Indirect(reflect.ValueOf(val))
		if stype.Kind() == reflect.Struct {
			field := stype.FieldByName("FS")
			if field.IsValid() {
				field.Set(reflect.ValueOf(f.deps.FS))
			}
		}
	}

	helpText := bytes.NewBufferString("")
	parser.WriteHelp(helpText)

	_, err := parser.ParseArgs(args)

	if boshOpts.UsernameOpt != "" {
		return Cmd{}, errors.New("BOSH_USER is deprecated use BOSH_CLIENT instead")
	}

	// --help and --version result in errors; turn them into successful output cmds
	if typedErr, ok := err.(*goflags.Error); ok {
		if typedErr.Type == goflags.ErrHelp {
			cmdOpts = &MessageOpts{Message: typedErr.Message}
			err = nil
		}
	}

	if _, ok := cmdOpts.(*HelpOpts); ok {
		cmdOpts = &MessageOpts{Message: helpText.String()}
	}

	return NewCmd(*boshOpts, cmdOpts, f.deps, f.compl), err
}
