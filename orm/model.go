package orm

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"gnd.la/orm/driver"
	"gnd.la/orm/index"
	"gnd.la/orm/query"
)

type JoinType int

const (
	JoinTypeInner JoinType = JoinType(driver.InnerJoin)
	JoinTypeOuter JoinType = JoinType(driver.OuterJoin)
	JoinTypeLeft  JoinType = JoinType(driver.LeftJoin)
	JoinTypeRight JoinType = JoinType(driver.RightJoin)
)

func (j JoinType) String() string {
	switch j {
	case JoinTypeInner:
		return "INNER JOIN"
	case JoinTypeOuter:
		return "OUTER JOIN"
	case JoinTypeLeft:
		return "LEFT OUTER JOIN"
	case JoinTypeRight:
		return "OUTER JOIN"
	}
	return "unknown JoinType"
}

const (
	modelSep = '|'
)

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
	namedReferences map[string]*model
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
			dir := v.Value("index")
			if dir == "" || dir == "asc" || dir == "both" {
				indexes = append(indexes, &index.Index{
					Fields: []string{m.fields.QNames[ii]},
					Unique: v.Has("unique"),
				})
			}
			if dir == "desc" || dir == "both" {
				name := m.fields.QNames[ii]
				idx := &index.Index{
					Fields: []string{name},
					Unique: v.Has("unique"),
				}
				indexes = append(indexes, idx.Set(index.DESC, name))
			}
		}
	}
	return indexes
}

func (m *model) Map(qname string) (string, reflect.Type, error) {
	if n, ok := m.fields.QNameMap[qname]; ok {
		return m.fields.QuotedNames[n], m.fields.Types[n], nil
	}
	if qname != "" && unicode.IsLower(rune(qname[0])) {
		return qname, nil, nil
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
	return m.name + string(modelSep) + qname
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
	if j.skip {
		s = append(s, "(Skipped)")
	}
	if j.join != nil {
		s = append(s, " JOIN ")
		s = append(s, j.join.model.String())
		s = append(s, " ON ")
		s = append(s, fmt.Sprintf("%+v", j.join.q))
	}
	return strings.Join(s, "")
}

func (j *joinModel) isJoinedWith(m *model) bool {
	for cur := j; cur != nil; cur = cur.Next() {
		if cur.model == m {
			return true
		}
	}
	return false
}

func (j *joinModel) Methods() []*driver.Methods {
	var methods []*driver.Methods
	for cur := j; cur != nil; cur = cur.Next() {
		methods = append(methods, cur.model.fields.Methods)
	}
	return methods
}

func (j *joinModel) joinWith(model *model, q query.Q, jt JoinType) (*joinModel, error) {
	if j.model == nil {
		j.model = model
		return j, nil
	}
	for cur := j; cur != nil; cur = cur.Next() {
		if cur.model == model {
			cur.skip = false
			return cur, nil
		}
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
		if len(candidates) > 1 {
			// Check if all the candidates point to the same
			// model and field. In that case, pick the first one.
			first := candidates[0]
			if eq, ok := first.q.(*query.Eq); ok {
				equal := true
				for _, v := range candidates[1:] {
					if veq, ok := v.q.(*query.Eq); !ok || first.model.model != v.model.model || !reflect.DeepEqual(eq.Value, veq.Value) {
						equal = false
						break
					}
				}
				if equal {
					candidates = candidates[:1]
				}
			}
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

func (j *joinModel) joinWithField(field string, jt JoinType) error {
	sep := strings.IndexByte(field, modelSep)
	if sep < 0 {
		return nil
	}
	typ := field[:sep]
	rem := field[sep+1:]
	m := j
	for {
		if model := m.model.namedReferences[typ]; model != nil {
			// Check if we're already joined to this model
			if j.isJoinedWith(model) {
				break
			}
			// Joins derived from queries are always implicit
			// and skipped, since we're only joining to check
			// against the value of the joined model.
			last, err := j.joinWith(model, nil, jt)
			if err != nil {
				return err
			}
			last.skip = true
			break
		}
		join := m.join
		if join == nil {
			break
		}
		m = join.model
	}
	if rem != "" {
		return m.joinWithField(rem, jt)
	}
	return nil
}

func (j *joinModel) joinWithSort(sort []driver.Sort, jt JoinType) error {
	for _, v := range sort {
		if err := j.joinWithField(v.Field(), jt); err != nil {
			return err
		}
	}
	return nil
}

func (j *joinModel) joinWithQuery(q query.Q, jt JoinType) error {
	if err := j.joinWithField(q.FieldName(), jt); err != nil {
		return err
	}
	for _, sq := range q.SubQ() {
		if err := j.joinWithQuery(sq, jt); err != nil {
			return err
		}
	}
	return nil
}

func (j *joinModel) Next() *joinModel {
	if j.join != nil {
		return j.join.model
	}
	return nil
}

func (j *joinModel) Map(qname string) (string, reflect.Type, error) {
	var candidates []mapCandidate
	parts := strings.Split(qname, string(modelSep))
	var field string
	var typ string
	switch len(parts) {
	case 1:
		field = parts[0]
	default:
		field = parts[len(parts)-1]
		typ = parts[len(parts)-2]
	}
	for cur := j; cur != nil; cur = cur.Next() {
		if typ != "" && typ != cur.model.name && typ != cur.model.shortName {
			continue
		}
		n, t, err := cur.model.Map(field)
		if err == nil {
			candidates = append(candidates, mapCandidate{n, t})
		}
	}
	switch len(candidates) {
	case 0:
		return "", nil, errCantMap(qname)
	case 1:
		c := candidates[0]
		return c.name, c.typ, nil
	default:
		return "", nil, &errAmbiguous{
			Field: qname,
			Model: j,
		}
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

type errAmbiguous struct {
	Field string
	Model *joinModel
}

func (e errAmbiguous) Error() string {
	var names []string
	for cur := e.Model; cur != nil; cur = cur.Next() {
		names = append(names, cur.model.name)
	}
	return fmt.Sprintf("field name %q is ambiguous (candidates are %v) - please, indicate the type like e.g. Type%sField",
		e.Field, strings.Join(names, ","), string(modelSep))
}
