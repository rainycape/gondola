package orm

import (
	"fmt"
	"gnd.la/orm/driver"
	"gnd.la/orm/index"
	"gnd.la/orm/query"
	"reflect"
	"strings"
)

type JoinType int

const (
	InnerJoin JoinType = JoinType(driver.InnerJoin)
	OuterJoin JoinType = JoinType(driver.OuterJoin)
	LeftJoin  JoinType = JoinType(driver.LeftJoin)
	RightJoin JoinType = JoinType(driver.RightJoin)
)

func (j JoinType) String() string {
	switch j {
	case InnerJoin:
		return "INNER JOIN"
	case OuterJoin:
		return "OUTER JOIN"
	case LeftJoin:
		return "LEFT OUTER JOIN"
	case RightJoin:
		return "OUTER JOIN"
	}
	return "unknown JoinType"
}

type reference struct {
	model string
	field string
}

type model struct {
	options         *Options
	name            string
	shortName       string
	table           string
	fields          *driver.Fields
	tags            string
	references      map[string]*reference
	modelReferences map[*model][]*join
}

func (m *model) Type() reflect.Type {
	return m.fields.Type
}

func (m *model) Table() string {
	return m.table
}

func (m *model) Fields() *driver.Fields {
	return m.fields
}

func (m *model) Indexes() []*index.Index {
	var indexes []*index.Index
	if m.options != nil {
		indexes = append(indexes, m.options.Indexes...)
	}
	// Add indexes declared in the fields
	for ii, v := range m.fields.Tags {
		if v.Has("index") {
			indexes = append(indexes, &index.Index{
				Fields: []string{m.fields.QNames[ii]},
				Unique: v.Has("unique"),
			})
		}
	}
	return indexes
}

func (m *model) Map(qname string) (string, reflect.Type, error) {
	sep := strings.IndexByte(qname, '|')
	if sep >= 0 {
		name := qname[:sep]
		if name != m.name && name != m.shortName {
			return "", nil, errNotThisModel(name)
		}
		qname = qname[sep+1:]
	}
	if n, ok := m.fields.QNameMap[qname]; ok {
		return m.fields.QuotedNames[n], m.fields.Types[n], nil
	}
	return "", nil, errCantMap(qname)
}

func (m *model) Skip() bool {
	return false
}

func (m *model) Join() driver.Join {
	return nil
}

func (m *model) String() string {
	return m.name
}

func (m *model) fullName(qname string) string {
	return m.name + "|" + qname
}

type join struct {
	model *joinModel
	jtype JoinType
	q     query.Q
}

func (j *join) Model() driver.Model {
	return j.model
}

func (j *join) Type() driver.JoinType {
	return driver.JoinType(j.jtype)
}

func (j *join) Query() query.Q {
	return j.q
}

func (j *join) clone() *join {
	return &join{
		model: j.model.clone(),
		jtype: j.jtype,
		q:     j.q,
	}
}

type joinModel struct {
	*model
	skip bool
	join *join
}

func (j *joinModel) clone() *joinModel {
	nj := &joinModel{
		model: j.model,
		skip:  j.skip,
	}
	if j.join != nil {
		nj.join = j.join.clone()
	}
	return nj
}

func (j *joinModel) Fields() *driver.Fields {
	if j.skip {
		return nil
	}
	return j.model.Fields()
}

func (j *joinModel) Skip() bool {
	return j.skip
}

func (j *joinModel) Join() driver.Join {
	// This workarounds a gotcha in Go which
	// generates an interface which points to nil
	// when returning a nil variable, thus making
	// the caller think it got a non-nil object if
	// it just checks for x != nil. The caller can
	// check for this using reflect, but it seems
	// easier and less error prone to circumvent the
	// problem right here.
	if j.join == nil {
		return nil
	}
	return j.join
}

func (j *joinModel) String() string {
	s := []string{j.model.name}
	for cur := j; cur.join != nil; cur = cur.join.model {
		s = append(s, " JOIN ")
		s = append(s, cur.join.model.name)
		s = append(s, " ON ")
		s = append(s, fmt.Sprintf("%+v", cur.join.q))
	}
	return strings.Join(s, "")
}

func (j *joinModel) joinWith(model *model, q query.Q, jt JoinType) (*joinModel, error) {
	if j.model == nil {
		j.model = model
		return j, nil
	}
	m := j
	if q == nil {
		var candidates []*join
		// Implicit join
		for {
			candidates = append(candidates, m.modelReferences[model]...)
			if m.join == nil {
				break
			}
			m = m.join.model
		}
		switch len(candidates) {
		case 1:
			m.join = candidates[0].clone()
			m.join.jtype = jt
		case 0:
			return nil, fmt.Errorf("can't join %s with model %s", j, model)
		default:
			return nil, fmt.Errorf("joining %s with model %s is ambiguous using query %+v", j, model, q)
		}
	} else {
		for m.join != nil {
			m = m.join.model
		}
		m.join = &join{
			model: &joinModel{model: model},
			jtype: jt,
			q:     q,
		}
	}
	return m.join.model, nil
}

func (j *joinModel) Map(qname string) (string, reflect.Type, error) {
	var candidates []mapCandidate
	for cur := j; ; {
		n, t, err := cur.model.Map(qname)
		if err == nil {
			candidates = append(candidates, mapCandidate{n, t})
		}
		if cur.join == nil {
			break
		}
		cur = cur.join.model
	}
	switch len(candidates) {
	case 0:
		return "", nil, errCantMap(qname)
	case 1:
		c := candidates[0]
		return c.name, c.typ, nil
	default:
		return "", nil, errAmbiguous(qname)
	}
	panic("unreachable")
}

type mapCandidate struct {
	name string
	typ  reflect.Type
}

type sortModels []driver.Model

func (s sortModels) Len() int {
	return len(s)
}

func (s sortModels) less(mi, mj driver.Model) bool {
	for _, v := range mi.Fields().References {
		if v.Model == mj {
			return false
		}
		if v.Model != mi && !s.less(v.Model, mj) {
			return false
		}
	}
	return true
}

func (s sortModels) Less(i, j int) bool {
	return s.less(s[i], s[j])
}

func (s sortModels) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type errCantMap string

func (e errCantMap) Error() string {
	return fmt.Sprintf("can't map field %q to a database name", string(e))
}

type errNotThisModel string

func (e errNotThisModel) Error() string {
	return fmt.Sprintf("name %q does not correspond to this model", string(e))
}

type errAmbiguous string

func (e errAmbiguous) Error() string {
	return fmt.Sprintf("field name %q is ambiguous. Please, indicate the type like e.g. Type|Field", string(e))
}
