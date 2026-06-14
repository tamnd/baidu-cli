package cli

import (
	"github.com/spf13/cobra"
)

// hotCmd fetches the Baidu hot search board.
func (a *App) hotCmd() *cobra.Command {
	var tab string

	cmd := &cobra.Command{
		Use:   "hot",
		Short: "Baidu hot search board",
		Long: `Fetch the Baidu hot search board for the given tab.

Available tabs: realtime, novel, movie, teleplay, car.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(30)
			a.progressf("fetching baidu hot search (%s)...", tab)
			items, err := a.client.Hot(cmd.Context(), tab, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(items, len(items))
		},
	}

	cmd.Flags().StringVar(&tab, "tab", "realtime", "board tab: realtime|novel|movie|teleplay|car")

	return cmd
}

// suggestCmd fetches Baidu search suggestions.
func (a *App) suggestCmd() *cobra.Command {
	var query string

	cmd := &cobra.Command{
		Use:   "suggest",
		Short: "Baidu search suggestions",
		Long:  `Fetch up to 10 query suggestions from Baidu for the given search term.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if query == "" {
				return codeError(exitUsage, nil)
			}
			a.progressf("fetching suggestions for %q...", query)
			suggestions, err := a.client.Suggest(cmd.Context(), query)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(suggestions, len(suggestions))
		},
	}

	cmd.Flags().StringVarP(&query, "query", "Q", "", "search term to get suggestions for (required)")
	_ = cmd.MarkFlagRequired("query")

	return cmd
}
