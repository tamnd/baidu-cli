// Package baidu provides typed data models for the baidu command.
package baidu

import "time"

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

// SearchResult is a single Baidu Search SERP entry.
type SearchResult struct {
	ID         string    `json:"id"`
	Query      string    `json:"query"`
	Page       int       `json:"page"`
	Position   int       `json:"position"`
	URL        string    `json:"url"`
	DisplayURL string    `json:"display_url"`
	Title      string    `json:"title"`
	Snippet    string    `json:"snippet" table:",truncate"`
	Tpl        string    `json:"tpl"`
	IsAd       bool      `json:"is_ad"`
	FetchedAt  time.Time `json:"fetched_at"`
}

// BaikeArticle is a Baidu Baike encyclopedia article. It is the centerpiece
// record, filled from the card API and the article HTML.
type BaikeArticle struct {
	LemmaID         int           `json:"lemma_id"`
	URL             string        `json:"url"`
	Title           string        `json:"title"`
	Subtitle        string        `json:"subtitle"`
	Abstract        string        `json:"abstract" table:",truncate"`
	BodyMarkdown    string        `json:"body_markdown" table:"-"`
	Infobox         []InfoboxItem `json:"infobox" table:"-"`
	Sections        []Section     `json:"sections" table:"-"`
	Categories      []string      `json:"categories"`
	Tags            []string      `json:"tags"`
	Synonyms        []string      `json:"synonyms"`
	RelatedIDs      []int         `json:"related_ids" table:"-"`
	Images          []ImageItem   `json:"images" table:"-"`
	ImageURL        string        `json:"image_url"`
	ReferencesCount int           `json:"references_count"`
	// Metadata from window.PAGE_DATA
	Creator      string    `json:"creator"`
	LastEditor   string    `json:"last_editor"`
	LastEditTime time.Time `json:"last_edit_time"`
	VersionID    int64     `json:"version_id"`
	FetchedAt    time.Time `json:"fetched_at"`
}

// ImageItem is an image found in a Baike article body.
type ImageItem struct {
	URL     string `json:"url"`
	Caption string `json:"caption"`
}

// InfoboxItem is one name/value pair of a Baike article infobox.
type InfoboxItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Section is one heading in a Baike article body.
type Section struct {
	Level int    `json:"level"`
	Name  string `json:"name"`
}

// CategoryLemma is a lightweight discovery record: one lemma found under a Baike
// category tag.
type CategoryLemma struct {
	LemmaID  int    `json:"lemma_id"`
	Title    string `json:"title"`
	Category string `json:"category"`
}

// wire types ─────────────────────────────────────────────────────────────────

type wireBoard struct {
	Data wireData `json:"data"`
}

type wireData struct {
	Cards []wireCard `json:"cards"`
}

type wireCard struct {
	Content []wireGroup `json:"content"`
}

// wireGroup is one entry under a card's content. The realtime board nests a
// second level: data.cards[].content[] holds groups, and each group carries its
// own content[] of items. The other boards (novel, movie, ...) put the item
// fields directly at this level. wireGroup embeds wireItem (the flat shape) and
// also carries a nested Content (the realtime shape), so a single walk handles
// either: prefer the nested items when present, else treat the group as an item.
type wireGroup struct {
	wireItem
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

// baikeAPIResp mirrors the Baike JSON card API response. A non-zero Errno is the
// card API's error path; from anti-bot-walled IPs it answers every key with
// {"errno":2} instead of card data.
type baikeAPIResp struct {
	Errno      int    `json:"errno"`
	ID         int    `json:"id"`
	NewLemmaID int    `json:"newLemmaId"`
	Key        string `json:"key"`
	Title      string `json:"title"`
	Desc       string `json:"desc"`
	Abstract   string `json:"abstract"`
	Image      string `json:"image"`
	URL        string `json:"url"`
	WapURL     string `json:"wapUrl"`
	HasOther   int    `json:"hasOther"`
	Card       []struct {
		Key    string   `json:"key"`
		Name   string   `json:"name"`
		Value  []string `json:"value"`
		Format []string `json:"format"`
	} `json:"card"`
}

// pageDataResp mirrors the window.PAGE_DATA JSON embedded in Baike article pages.
type pageDataResp struct {
	LemmaID           int    `json:"lemmaId"`
	LemmaTitle        string `json:"lemmaTitle"`
	Uname             string `json:"uname"`
	CreateTime        int64  `json:"createTime"`
	UpdateTime        int64  `json:"updateTime"`
	VersionID         int64  `json:"versionId"`
	VersionUname      string `json:"versionUname"`
	VersionCreateTime int64  `json:"versionCreateTime"`
	ExtData           struct {
		Classify []struct {
			Path string `json:"path"`
			Name string `json:"name"`
		} `json:"classify"`
		KpdClassify []struct {
			Name string `json:"name"`
		} `json:"kpdClassify"`
		RedirectFrom []struct {
			FromTitle string `json:"fromTitle"`
		} `json:"redirectFrom"`
		Pinyin string `json:"pinyin"`
	} `json:"extData"`
}

// wikitagLemma is one item from wikitag/api/getlemmas.
type wikitagLemma struct {
	LemmaID int    `json:"lemma_id"`
	Title   string `json:"title"`
	After   string `json:"after"`
}
