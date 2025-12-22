package feishu

import "fmt"

// Message 消息接口
type Message interface {
	Type() string
	Content() interface{}
}

// TextMessage 文本消息
type TextMessage struct {
	text string
}

// NewTextMessage 创建文本消息
func NewTextMessage(text string) *TextMessage {
	return &TextMessage{text: text}
}

func (m *TextMessage) Type() string {
	return "text"
}

func (m *TextMessage) Content() interface{} {
	return map[string]interface{}{
		"text": m.text,
	}
}

// PostMessage 富文本消息（支持 @ 用户、链接等）
type PostMessage struct {
	title   string
	content [][]MessageElement
}

// MessageElement 消息元素
type MessageElement struct {
	Tag    string `json:"tag"`
	Text   string `json:"text,omitempty"`
	UserID string `json:"user_id,omitempty"`
	Href   string `json:"href,omitempty"`
}

// NewPostMessage 创建富文本消息
func NewPostMessage(title string) *PostMessage {
	return &PostMessage{
		title:   title,
		content: [][]MessageElement{},
	}
}

// AddLine 添加一行内容（可包含多个元素）
func (m *PostMessage) AddLine(elements ...MessageElement) *PostMessage {
	m.content = append(m.content, elements)
	return m
}

func (m *PostMessage) Type() string {
	return "post"
}

func (m *PostMessage) Content() interface{} {
	return map[string]interface{}{
		"post": map[string]interface{}{
			"zh_cn": map[string]interface{}{
				"title":   m.title,
				"content": m.content,
			},
		},
	}
}

// 辅助函数：创建消息元素

// Text 创建文本元素
func Text(text string) MessageElement {
	return MessageElement{Tag: "text", Text: text}
}

// At 创建 @ 用户元素
func At(userID string) MessageElement {
	return MessageElement{Tag: "at", UserID: userID}
}

// Link 创建链接元素
func Link(text, href string) MessageElement {
	return MessageElement{Tag: "a", Text: text, Href: href}
}

// AtAll 创建 @ 所有人元素
func AtAll() MessageElement {
	return MessageElement{Tag: "at", UserID: "all"}
}

// Bold 创建加粗文本（使用文本标记）
func Bold(text string) MessageElement {
	return MessageElement{Tag: "text", Text: fmt.Sprintf("**%s**", text)}
}
