// Copyright (c) 2024 John Millikin <john@john-millikin.com>
//
// Permission to use, copy, modify, and/or distribute this software for any
// purpose with or without fee is hereby granted.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH
// REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY
// AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT,
// INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM
// LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR
// OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR
// PERFORMANCE OF THIS SOFTWARE.
//
// SPDX-License-Identifier: 0BSD

package main

import (
	"context"
	stdflag "flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type command interface {
	help() *commandHelp
	flags(flags *pflag.FlagSet)
	run(ctx context.Context, argv []string) int
}

type commandHelp struct {
	usage   string
	summary string
}

func main() {
	ctx := context.Background()

	idolCmd := &cobra.Command{
		Use: "idol [options] COMMAND",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}
	idolCmd.RunE = func(cmd *cobra.Command, args []string) error {
		fmt.Fprint(os.Stderr, idolCmd.UsageString())
		os.Exit(1)
		return nil
	}

	commands := []command{
		&cmdCompile{},
		&cmdCodegen{},
		&cmdFormat{},
	}
	for _, cmd := range commands {
		help := cmd.help()
		cobraCmd := &cobra.Command{
			Use:   help.usage,
			Short: help.summary,
			RunE: func(_ *cobra.Command, args []string) error {
				os.Exit(cmd.run(ctx, args))
				return nil
			},
		}
		idolCmd.AddCommand(cobraCmd)
		cmd.flags(cobraCmd.Flags())
	}

	idolCmd.Flags().AddGoFlagSet(stdflag.CommandLine)
	idolCmd.ParseFlags(nil)
	if _, err := idolCmd.ExecuteC(); err != nil {
		os.Exit(1)
	}
}
