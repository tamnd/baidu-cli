package baidu

import "context"

// The input structs below are reflected by kit to build CLI flags, HTTP query
// params, and MCP tool arguments. Each carries the injected *Client and, where a
// command has a default fetch count, an inherited --limit so the handler can
// size its request. kit enforces the row limit again around emit.

type hotIn struct {
	Tab    string  `kit:"flag,short=t" default:"realtime" enum:"realtime,novel,movie,teleplay,car" help:"hot board tab"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func hot(ctx context.Context, in hotIn, emit func(HotItem) error) error {
	items, err := in.Client.Hot(ctx, in.Tab, limitOr(in.Limit, 30))
	if err != nil {
		return mapErr(err)
	}
	for _, it := range items {
		if err := emit(it); err != nil {
			return err
		}
	}
	return nil
}

type suggestIn struct {
	Query  string  `kit:"arg" help:"search query"`
	Client *Client `kit:"inject"`
}

func suggest(ctx context.Context, in suggestIn, emit func(Suggestion) error) error {
	items, err := in.Client.Suggest(ctx, in.Query)
	if err != nil {
		return mapErr(err)
	}
	for _, s := range items {
		if err := emit(s); err != nil {
			return err
		}
	}
	return nil
}

type searchIn struct {
	Query  string  `kit:"arg" help:"search query"`
	Pages  int     `kit:"flag" default:"1" help:"number of result pages to fetch"`
	Client *Client `kit:"inject"`
}

func search(ctx context.Context, in searchIn, emit func(SearchResult) error) error {
	pages := in.Pages
	if pages < 1 {
		pages = 1
	}
	results, err := in.Client.Search(ctx, in.Query, pages)
	if err != nil {
		return mapErr(err)
	}
	for _, r := range results {
		if err := emit(r); err != nil {
			return err
		}
	}
	return nil
}

type articleIn struct {
	Ref    string  `kit:"arg" help:"lemma id, title, or title/id"`
	Client *Client `kit:"inject"`
}

func article(ctx context.Context, in articleIn, emit func(*BaikeArticle) error) error {
	art, err := in.Client.Article(ctx, in.Ref)
	if err != nil {
		return mapErr(err)
	}
	return emit(art)
}

type categoriesIn struct {
	Tag    string  `kit:"arg" help:"category tag (empty walks all 16 built-in tags)"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func categories(ctx context.Context, in categoriesIn, emit func(CategoryLemma) error) error {
	lemmas, err := in.Client.Categories(ctx, in.Tag, limitOr(in.Limit, 40))
	if err != nil {
		return mapErr(err)
	}
	for _, l := range lemmas {
		if err := emit(l); err != nil {
			return err
		}
	}
	return nil
}
