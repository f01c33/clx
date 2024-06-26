package cmd

import (
	_ "embed"
	"strconv"
	"time"

	"github.com/bensadeh/circumflex/less"

	"github.com/bensadeh/circumflex/hn/services/hybrid"

	"github.com/bensadeh/circumflex/cli"
	"github.com/bensadeh/circumflex/screen"
	"github.com/bensadeh/circumflex/tree"

	"github.com/spf13/cobra"
)

func commentsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "comments",
		Short: "Go directly to the comment section by ID",
		Long: "Directly enter the comment section for a given item without going through the main " +
			"view first",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			id, _ := strconv.Atoi(args[0])

			service := new(hybrid.Service)

			comments := service.FetchComments(id)

			config := getConfig()

			screenWidth := screen.GetTerminalWidth()
			commentTree := tree.Print(comments, config, screenWidth, time.Now().Unix())

			lesskey := less.NewLesskey()
			config.LesskeyPath = lesskey.GetPath()

			command := cli.Less(commentTree, config)

			if err := command.Run(); err != nil {
				defer lesskey.Remove()
				panic(err)
			}
		},
	}
}
