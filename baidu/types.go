// Package baidu provides typed data models for the baidu command.
package baidu

// HotItem is one entry from the Baidu hot search board.
type HotItem struct {
	Rank int    `json:"rank"`
	Word string `json:"word"`
	Tag  string `json:"tag"`
	URL  string `json:"url"`
}

// Suggestion is one entry from the Baidu suggest API.
type Suggestion struct {
	Rank int    `json:"rank"`
	Word string `json:"word"`
}

// wire types ─────────────────────────────────────────────────────────────────

type wireBoard struct {
	Data wireData `json:"data"`
}

type wireData struct {
	Cards []wireCard `json:"cards"`
}

type wireCard struct {
	Content []wireItem `json:"content"`
}

type wireItem struct {
	IsTop      bool   `json:"isTop"`
	Index      int    `json:"index"`
	Word       string `json:"word"`
	URL        string `json:"url"`
	HotTag     string `json:"hotTag"`
	NewHotName string `json:"newHotName"`
}

func wireToHotItem(w wireItem) HotItem {
	rank := 0
	if !w.IsTop {
		rank = w.Index + 1
	}
	tag := w.NewHotName
	if tag == "" && w.HotTag == "3" {
		tag = "热"
	}
	if tag == "" && w.HotTag == "1" {
		tag = "新"
	}
	return HotItem{
		Rank: rank,
		Word: w.Word,
		Tag:  tag,
		URL:  w.URL,
	}
}
