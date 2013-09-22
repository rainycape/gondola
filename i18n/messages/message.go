package messages

import (
	"fmt"
	"go/token"
	"gnd.la/astutil"
	"sort"
)

type Position struct {
	Filename string
	Line     int
	Comment  string
}

func (p *Position) String() string {
	return fmt.Sprintf("%s:%d", p.Filename, p.Line)
}

type positions []*Position

func (p positions) Len() int {
	return len(p)
}

func (p positions) Less(i, j int) bool {
	if p[i].Filename < p[j].Filename {
		return true
	}
	if p[i].Filename > p[j].Filename {
		return false
	}
	// Same file
	return p[i].Line < p[j].Line
}

func (p positions) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type Message struct {
	Context           string
	Singular          string
	Plural            string
	TranslatorComment string
	Positions         []*Position
	Translations      []string
}

func (m *Message) Key() string {
	return m.Context + m.Singular
}

func (m *Message) Merge(o *Message, pos *Position) error {
	if m.Plural != "" && m.Plural != "" && m.Plural != o.Plural {
		return fmt.Errorf("different plural forms for singular form %q: %q (%v) and %q (%v)", m.Singular,
			m.Plural, m.Positions, o.Plural, o.Positions)
	}
	if m.Plural == "" {
		m.Plural = o.Plural
	}
	m.Positions = append(m.Positions, pos)
	sort.Sort(positions(m.Positions))
	return nil
}

func (m *Message) String() string {
	return fmt.Sprintf("%q", m.Singular)
}

type messageSlice []*Message

func (m messageSlice) Len() int {
	return len(m)
}

func (m messageSlice) Less(i, j int) bool {
	if m[i].Singular < m[j].Singular {
		return true
	}
	if m[i].Singular > m[j].Singular {
		return false
	}
	return m[i].Context < m[j].Context
}

func (m messageSlice) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

type messageMap map[string]*Message

func (m messageMap) Add(msg *Message, pos *token.Position, comment string) error {
	p := &Position{pos.Filename, pos.Line, comment}
	k := msg.Key()
	if message, ok := m[k]; ok {
		if err := message.Merge(msg, p); err != nil {
			return err
		}
	} else {
		msg.Positions = []*Position{p}
		m[k] = msg
	}
	return nil
}

func (m messageMap) AddString(s *astutil.String, comment string) error {
	message := &Message{
		Context:  s.Context(),
		Singular: s.Singular(),
		Plural:   s.Plural(),
	}
	if err := m.Add(message, s.Position, comment); err != nil {
		return err
	}
	return nil
}

func (m messageMap) Messages() []*Message {
	messages := make(messageSlice, len(m))
	ii := 0
	for _, v := range m {
		messages[ii] = v
		ii++
	}
	sort.Sort(messages)
	return ([]*Message)(messages)
}
