package markdown

// Kind identifies a block-level AST node. The AST is renderer independent: HTML,
// PDF and DOCX exporters consume the same tree instead of reparsing markdown.
type Kind uint8

const (
	Document Kind = iota
	Heading
	Paragraph
	BulletList
	OrderedList
	ListItem
	CodeBlock
	BlockQuote
	HorizontalRule
	Table
	Alert
	HTMLBlock
	DefinitionList
	DefinitionTerm
	DefinitionItem
	FootnoteList
	FootnoteDefinition
)

type InlineKind uint8

const (
	InlineText InlineKind = iota
	InlineEmphasis
	InlineStrong
	InlineCode
	InlineLink
	InlineImage
	InlineStrike
	InlineSoftBreak
	InlineHardBreak
	InlineHTML
)

type Align uint8

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

type Position struct {
	Line   int
	Column int
	Offset int
}

type Inline struct {
	Kind     InlineKind
	Text     string
	Dest     string
	Title    string
	Children []Inline
	Start    Position
	End      Position
}

type Node struct {
	Kind     Kind
	Level    int
	Text     string
	Info     string
	ID       string
	Checked  *bool
	Children []Node
	Rows     [][]string
	Aligns   []Align
	Attrs    map[string]string
	Start    Position
	End      Position
	Inlines  []Inline
}

type HeadingRef struct {
	Level int
	Text  string
	ID    string
}

type LinkRef struct {
	URL   string
	Title string
}

type Footnote struct {
	ID    string
	Text  string
	Nodes []Node
}

type Warning struct {
	Line    int
	Message string
}

type Doc struct {
	Nodes     []Node
	Meta      map[string]string
	Headings  []HeadingRef
	Refs      map[string]LinkRef
	Footnotes []Footnote
	Warnings  []Warning
}

type Options struct {
	MaxLineBytes int
	Strict       bool
	UnsafeHTML   bool
}
