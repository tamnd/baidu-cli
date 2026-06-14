package baidu

import "errors"

const (
	// BaikeBaseURL is the encyclopedia host.
	BaikeBaseURL = "https://baike.baidu.com"
	// SearchBaseURL is the web search host (also serves the suggest sugrec path).
	SearchBaseURL = "https://www.baidu.com"

	// EntityLemma is a Baike article frontier row.
	EntityLemma = "lemma"
	// EntitySERP is a web search results frontier row.
	EntitySERP = "serp"
	// EntitySuggest is a suggest query record source.
	EntitySuggest = "suggest"
)

// ErrNotFound is the sentinel for a missing lemma so the kit layer can map it to
// the not-found / no-results exit codes.
var ErrNotFound = errors.New("not found")

// ErrBlocked is the sentinel for a walled SERP (CAPTCHA or block page) so the
// kit layer can map it to the rate-limited exit and surface the --baiduid hint.
var ErrBlocked = errors.New("blocked by Baidu (CAPTCHA or rate limit)")

// userAgents is the rotation pool for the search and baike fetchers. A real
// browser fingerprint helps the SERP, and the pool keeps requests from looking
// like a single probe.
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:135.0) Gecko/20100101 Firefox/135.0",
}

// BaikeSeedTopics is the bootstrap list of seed topics for breadth-first
// discovery. Each is resolved through the card API and, if it resolves, the
// article HTML is enqueued for the full body.
var BaikeSeedTopics = []string{
	// Technology
	"Python", "Java", "Go语言", "C++", "JavaScript", "人工智能", "机器学习",
	"深度学习", "神经网络", "大数据", "区块链", "云计算", "物联网",
	"操作系统", "Linux", "互联网", "数据库", "算法", "计算机",
	// Science
	"量子力学", "相对论", "DNA", "基因组", "宇宙", "黑洞", "进化论",
	"光合作用", "元素周期表", "引力波", "化学元素", "物理学", "数学",
	// Chinese History
	"中国历史", "中华人民共和国", "唐朝", "宋朝", "明朝", "清朝",
	"秦始皇", "汉朝", "丝绸之路", "改革开放", "抗日战争", "长征",
	// Geography
	"北京", "上海", "广州", "深圳", "长江", "黄河", "西藏",
	"新疆", "台湾", "喜马拉雅山", "珠穆朗玛峰", "黄土高原",
	// Culture
	"春节", "中秋节", "汉字", "普通话", "儒家思想", "道教", "佛教",
	"中医", "武术", "围棋", "书法", "中国画", "京剧",
	// People
	"毛泽东", "邓小平", "孔子", "老子", "李白", "杜甫",
	"鲁迅", "袁隆平", "屠呦呦",
	// Health
	"新冠病毒", "流感", "癌症", "高血压", "糖尿病", "疫苗",
	// Economy
	"GDP", "人民币", "通货膨胀", "一带一路", "股票市场",
	// Arts
	"电影", "音乐", "网络小说", "动漫",
}

// BaikeCategoryTags are the 16 top-level Baike category tags walked by the
// categories op and the mirror seeder.
var BaikeCategoryTags = []string{
	"历史", "科学", "地理", "人物", "文化", "技术", "艺术", "体育",
	"经济", "政治", "教育", "医学", "自然", "哲学", "法律", "军事",
}
