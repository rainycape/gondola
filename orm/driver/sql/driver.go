package sql

import (
	"bytes"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"gnd.la/app/profile"
	"gnd.la/config"
	"gnd.la/encoding/codec"
	"gnd.la/encoding/pipe"
	"gnd.la/log"
	"gnd.la/orm/driver"
	"gnd.la/orm/index"
	"gnd.la/orm/operation"
	"gnd.la/orm/query"
	"gnd.la/util/structs"
)

var (
	stringType = reflect.TypeOf("")
)

type Driver struct {
	db         *DB
	logger     *log.Logger
	backend    Backend
	transforms map[reflect.Type]struct{}
}

func (d *Driver) Check() error {
	if err := d.db.sqlDb.Ping(); err != nil {
		return err
	}
	return d.backend.Check(d.db)
}

func (d *Driver) Initialize(ms []driver.Model) error {
	// Create tables
	for _, v := range ms {
		tbl, err := d.makeTable(v)
		if err != nil {
			return err
		}
		existingTbl, err := d.backend.Inspect(d.db, v)
		if err != nil {
			return err
		}
		if existingTbl != nil {
			err = d.mergeTable(v, existingTbl, tbl)
		} else {
			if len(tbl.Fields) == 0 {
				log.Debugf("Skipping collection %s (model %v) because it has no fields", v.Table, v)
				continue
			}
			// Table does not exists, create it
			err = d.createTable(v, tbl)
		}
		if err != nil {
			return err
		}
	}
	// Create indexes
	for _, v := range ms {
		if err := d.createIndexes(v); err != nil {
			return err
		}
	}
	return nil
}

func (d *Driver) createIndexes(m driver.Model) error {
	for _, idx := range m.Indexes() {
		name, err := d.indexName(m, idx)
		if err != nil {
			return err
		}
		if err := d.createIndex(m, idx, name); err != nil {
			return err
		}
	}
	return nil
}

func (d *Driver) createIndex(m driver.Model, idx *index.Index, name string) error {
	has, err := d.backend.HasIndex(d.db, m, idx, name)
	if err != nil {
		return err
	}
	if has {
		return nil
	}

	buf := getBuffer()
	buf.WriteString("CREATE ")
	if idx.Unique {
		buf.WriteString("UNIQUE ")
	}
	buf.WriteString("INDEX IF NOT EXISTS ")
	buf.WriteString(name)
	buf.WriteString(" ON \"")
	buf.WriteString(m.Table())
	buf.WriteString("\" (")
	fields := m.Fields()
	for _, v := range idx.Fields {
		name, _, err := fields.Map(v)
		if err != nil {
			return err
		}
		buf.WriteString(name)
		if DescField(idx, v) {
			buf.WriteString(" DESC")
		}
		buf.WriteByte(',')
	}
	buf.Truncate(buf.Len() - 1)
	buf.WriteString(")")
	_, err = d.db.Exec(buftos(buf))
	putBuffer(buf)
	return err
}

func (d *Driver) indexName(m driver.Model, idx *index.Index) (string, error) {
	if len(idx.Fields) == 0 {
		return "", fmt.Errorf("index on %v has no fields", m.Type())
	}
	buf := getBuffer()
	buf.WriteString(m.Table())
	for _, v := range idx.Fields {
		dbName, _, err := m.Map(v)
		if err != nil {
			return "", err
		}
		buf.WriteByte('_')
		// dbName is quoted and includes the table name
		// extract the unquoted field name.
		buf.WriteString(unquote(dbName))
	}
	s := buf.String()
	putBuffer(buf)
	return s, nil
}

func (d *Driver) Query(m driver.Model, q query.Q, sort []driver.Sort, limit int, offset int) driver.Iter {
	query, params, err := d.Select(nil, true, m, q, sort, limit, offset)
	if err != nil {
		return &Iter{err: err}
	}
	rows, err := d.db.Query(bstos(query.Bytes()), params...)
	if err != nil {
		return &Iter{err: err}
	}
	return &Iter{model: m, rows: rows, driver: d}
}

func (d *Driver) Count(m driver.Model, q query.Q, limit int, offset int) (uint64, error) {
	var count uint64
	query, params, err := d.Select([]string{"COUNT(*)"}, false, m, q, nil, limit, offset)
	if err != nil {
		return 0, err
	}
	err = d.db.QueryRow(buftos(query), params...).Scan(&count)
	putBuffer(query)
	return count, err
}

func (d *Driver) Exists(m driver.Model, q query.Q) (bool, error) {
	query, params, err := d.Select([]string{"1"}, false, m, q, nil, -1, -1)
	if err != nil {
		return false, err
	}
	var one uint64
	err = d.db.QueryRow(buftos(query), params...).Scan(&one)
	putBuffer(query)
	if err == sql.ErrNoRows {
		err = nil
	}
	return one == 1, err
}

func (d *Driver) Insert(m driver.Model, data interface{}) (driver.Result, error) {
	_, fields, values, err := d.saveParameters(m, data)
	if err != nil {
		return nil, err
	}
	buf := getBuffer()
	buf.WriteString("INSERT INTO ")
	buf.WriteByte('"')
	buf.WriteString(m.Table())
	buf.WriteByte('"')
	count := len(fields)
	if count > 0 {
		buf.WriteString(" (")
		for _, v := range fields {
			buf.WriteByte('"')
			buf.WriteString(v)
			buf.WriteByte('"')
			buf.WriteByte(',')
		}
		buf.Truncate(buf.Len() - 1)
		buf.WriteString(") VALUES (")
		buf.WriteString(d.backend.Placeholders(count))
		buf.WriteByte(')')
	} else {
		buf.WriteByte(' ')
		buf.WriteString(d.backend.DefaultValues())
	}
	res, err := d.backend.Insert(d.db, m, buftos(buf), values...)
	putBuffer(buf)
	return res, err
}

func (d *Driver) Operate(m driver.Model, q query.Q, op *operation.Operation) (driver.Result, error) {
	dbName, _, err := m.Map(op.Field)
	if err != nil {
		return nil, err
	}
	dbName = unquote(dbName)
	buf := getBuffer()
	buf.WriteString("UPDATE ")
	buf.WriteByte('"')
	buf.WriteString(m.Table())
	buf.WriteByte('"')
	buf.WriteString(" SET ")
	buf.WriteString(dbName)
	buf.WriteByte('=')
	var params []interface{}
	switch op.Operator {
	case operation.OpAdd, operation.OpSub:
		buf.WriteString(dbName)
		if op.Operator == operation.OpAdd {
			buf.WriteByte('+')
		} else {
			buf.WriteByte('-')
		}
		buf.WriteString(d.backend.Placeholder(1))
		params = append(params, op.Value)
	case operation.OpSet:
		if f, ok := op.Value.(operation.Field); ok {
			fieldName, _, err := m.Map(string(f))
			if err != nil {
				return nil, err
			}
			buf.WriteString(unquote(fieldName))
		} else {
			buf.WriteString(d.backend.Placeholder(1))
			params = append(params, op.Value)
		}
	default:
		return nil, fmt.Errorf("operator %d is not supported", op.Operator)
	}
	qParams, err := d.where(buf, m, q, len(params))
	if err != nil {
		return nil, err
	}
	params = append(params, qParams...)
	res, err := d.db.Exec(buftos(buf), params...)
	putBuffer(buf)
	return res, err
}

func (d *Driver) Update(m driver.Model, q query.Q, data interface{}) (driver.Result, error) {
	_, fields, values, err := d.saveParameters(m, data)
	if err != nil {
		return nil, err
	}
	buf := getBuffer()
	buf.WriteString("UPDATE ")
	buf.WriteByte('"')
	buf.WriteString(m.Table())
	buf.WriteByte('"')
	buf.WriteString(" SET ")
	for ii, v := range fields {
		buf.WriteString(v)
		buf.WriteByte('=')
		buf.WriteString(d.backend.Placeholder(ii + 1))
		buf.WriteByte(',')
	}
	// remove last ,
	buf.Truncate(buf.Len() - 1)
	qParams, err := d.where(buf, m, q, len(values))
	if err != nil {
		return nil, err
	}
	params := append(values, qParams...)
	res, err := d.db.Exec(buftos(buf), params...)
	putBuffer(buf)
	return res, err
}

func (d *Driver) Upsert(m driver.Model, q query.Q, data interface{}) (driver.Result, error) {
	// TODO: MySql might be able to provide upserts
	return nil, nil
}

func (d *Driver) Delete(m driver.Model, q query.Q) (driver.Result, error) {
	buf := getBuffer()
	buf.WriteString("DELETE FROM ")
	buf.WriteByte('"')
	buf.WriteString(m.Table())
	buf.WriteByte('"')
	params, err := d.where(buf, m, q, 0)
	if err != nil {
		return nil, err
	}
	res, err := d.db.Exec(buftos(buf), params...)
	putBuffer(buf)
	return res, err
}

func (d *Driver) Close() error {
	return d.db.sqlDb.Close()
}

func (d *Driver) Upserts() bool {
	return false
}

func (d *Driver) Tags() []string {
	return []string{d.backend.Tag(), "sql"}
}

func (d *Driver) DB() *DB {
	return d.db
}

func (d *Driver) SetLogger(logger *log.Logger) {
	d.logger = logger
}

func (d *Driver) debugq(sql string, args []interface{}) {
	if profile.On && profile.Profiling() {
		if profile.HasEvent() {
			profile.Note("SQL: %s, args %v", sql, args)
			if strings.HasPrefix(sql, "SELECT") {
				rows, _ := d.db.sqlDb.Query("EXPLAIN "+sql, args...)
				if rows != nil {
					var line string
					var explain []string
					for rows.Next() {
						if err := rows.Scan(&line); err != nil {
							explain = nil
							break
						}
						explain = append(explain, line)
					}
					rows.Close()
					if len(explain) > 0 {
						profile.Note("%s", strings.Join(explain, "\n"))
					}
				}
			}
		}
	}
	if d.logger != nil {
		if len(args) > 0 {
			d.logger.Debugf("SQL: %s with arguments %v", sql, args)
		} else {
			d.logger.Debugf("SQL: %s", sql)
		}
	}
}

func (d *Driver) fieldByIndex(val reflect.Value, indexes []int, alloc bool) reflect.Value {
	for _, v := range indexes {
		if val.Type().Kind() == reflect.Ptr {
			if val.IsNil() {
				if !alloc {
					return reflect.Value{}
				}
				val.Set(reflect.New(val.Type().Elem()))
			}
			val = val.Elem()
		}
		val = val.Field(v)
	}
	return val
}

func (d *Driver) saveParameters(m driver.Model, data interface{}) (reflect.Value, []string, []interface{}, error) {
	// data is guaranteed to be of m.Type()
	val := driver.Direct(reflect.ValueOf(data))
	fields := m.Fields()
	max := len(fields.MNames)
	names := make([]string, 0, max)
	values := make([]interface{}, 0, max)
	var err error
	if d.transforms != nil {
		for ii, v := range fields.Indexes {
			f := d.fieldByIndex(val, v, false)
			if !f.IsValid() {
				continue
			}
			if fields.OmitEmpty[ii] && driver.IsZero(f) {
				continue
			}
			ft := f.Type()
			var fval interface{}
			if _, ok := d.transforms[ft]; ok {
				fval, err = d.backend.TransformOutValue(f)
				if err != nil {
					return val, nil, nil, err
				}
				if fields.NullEmpty[ii] && driver.IsZero(reflect.ValueOf(fval)) {
					fval = nil
				}
			} else if !fields.NullEmpty[ii] || !driver.IsZero(f) {
				if c := codec.FromTag(fields.Tags[ii]); c != nil {
					fval, err = c.Encode(f.Interface())
					if err != nil {
						return val, nil, nil, err
					}
					if p := pipe.FromTag(fields.Tags[ii]); p != nil {
						data, err := p.Encode(fval.([]byte))
						if err != nil {
							return val, nil, nil, err
						}
						fval = data
					}
				} else {
					// Most sql drivers won't accept aliases for string type
					if ft.Kind() == reflect.String && ft != stringType {
						f = f.Convert(stringType)
					}
					fval = f.Interface()
				}
			}
			names = append(names, fields.MNames[ii])
			values = append(values, fval)
		}
	} else {
		for ii, v := range fields.Indexes {
			f := d.fieldByIndex(val, v, false)
			if !f.IsValid() {
				continue
			}
			if fields.OmitEmpty[ii] && driver.IsZero(f) {
				continue
			}
			var fval interface{}
			if !fields.NullEmpty[ii] || !driver.IsZero(f) {
				if c := codec.FromTag(fields.Tags[ii]); c != nil {
					fval, err = c.Encode(&f)
					if err != nil {
						return val, nil, nil, err
					}
				} else {
					ft := f.Type()
					// Most sql drivers won't accept aliases for string type
					if ft.Kind() == reflect.String && ft != stringType {
						f = f.Convert(stringType)
					}
					fval = f.Interface()
				}
			}
			names = append(names, fields.MNames[ii])
			values = append(values, fval)
		}
	}
	return val, names, values, nil
}

func (d *Driver) outValues(m driver.Model, out interface{}) (reflect.Value, *driver.Fields, []interface{}, []*scanner, error) {
	val := reflect.ValueOf(out)
	if !val.IsValid() {
		// Untyped nil pointer
		return reflect.Value{}, nil, nil, nil, nil
	}
	vt := val.Type()
	if vt.Kind() != reflect.Ptr {
		return reflect.Value{}, nil, nil, nil, fmt.Errorf("can't set object of type %T. Please, pass a %v rather than a %v", out, reflect.PtrTo(vt), vt)
	}
	if vt.Elem().Kind() == reflect.Ptr && vt.Elem().Elem().Kind() == reflect.Struct {
		// Received a pointer to pointer. Always create a new object,
		// to avoid overwriting the previous result.
		val = val.Elem()
		el := reflect.New(val.Type().Elem())
		val.Set(el)
	}
	for val.Kind() == reflect.Ptr {
		el := val.Elem()
		if !el.IsValid() {
			if !val.CanSet() {
				// Typed nil pointer
				return reflect.Value{}, nil, nil, nil, nil
			}
			el = reflect.New(val.Type().Elem())
			val.Set(el)
		}
		val = el
	}
	fields := m.Fields()
	if fields == nil {
		// Skipped model
		return reflect.Value{}, nil, nil, nil, nil
	}
	values := make([]interface{}, len(fields.Indexes))
	scanners := make([]*scanner, len(fields.Indexes))
	for ii, v := range fields.Indexes {
		field := d.fieldByIndex(val, v, true)
		tag := fields.Tags[ii]
		s := newScanner(&field, tag, d.backend)
		scanners[ii] = s
		values[ii] = s
	}
	return val, fields, values, scanners, nil
}

func (d *Driver) isPrimaryKey(fields *driver.Fields, idx int, tag *structs.Tag) bool {
	if tag.Has("primary_key") {
		return true
	}
	for _, v := range fields.CompositePrimaryKey {
		if v == idx {
			return true
		}
	}
	return false
}

func (d *Driver) makeTable(m driver.Model) (*Table, error) {
	fields := m.Fields()
	names := fields.MNames
	qnames := fields.QNames
	ftypes := fields.Types
	tags := fields.Tags
	dbFields := make([]*Field, len(names))
	for ii, v := range names {
		typ := ftypes[ii]
		tag := tags[ii]
		ft, err := d.backend.FieldType(typ, tag)
		if err != nil {
			return nil, err
		}
		def := tag.Value("default")
		if fields.HasDefault(ii) {
			// Handled by the ORM
			def = ""
		}
		if def != "" {
			if driver.IsFunc(def) {
				fname, _ := driver.SplitFuncArgs(def)
				fn, err := d.backend.Func(fname, ftypes[ii])
				if err != nil {
					if err == ErrFuncNotSupported {
						err = fmt.Errorf("backend %s does not support function %s", d.backend.Name(), tag.Value("default"))
					}
					return nil, err
				}
				def = fn
			} else {
				def = driver.UnescapeDefault(def)
				if typ.Kind() == reflect.String {
					def = d.db.QuoteString(def)
				}
			}
		}
		field := &Field{
			Name:    v,
			Type:    ft,
			Default: def,
		}
		if tag.Has("notnull") {
			field.AddConstraint(ConstraintNotNull)
		}
		if d.isPrimaryKey(fields, ii, tag) {
			field.AddConstraint(ConstraintPrimaryKey)
		} else if tag.Has("unique") {
			field.AddConstraint(ConstraintUnique)
		}
		if tag.Has("auto_increment") {
			field.AddOption(OptionAutoIncrement)
		}
		if ref := fields.References[qnames[ii]]; ref != nil {
			fk, _, err := ref.Model.Fields().Map(ref.Field)
			if err != nil {
				return nil, err
			}
			field.Constraints = append(field.Constraints, &Constraint{
				Type:       ConstraintForeignKey,
				References: MakeReference(ref.Model.Table(), fk),
			})
		}
		dbFields[ii] = field
	}
	return &Table{Fields: dbFields}, nil
}

func (d *Driver) createTable(m driver.Model, table *Table) error {
	sql, err := table.SQL(d.db, d.backend, m, m.Table())
	if err != nil {
		return err
	}
	_, err = d.db.Exec(sql)
	return err
}

func (d *Driver) mergeTable(m driver.Model, prevTable *Table, newTable *Table) error {
	existing := make(map[string]*Field)
	for _, v := range prevTable.Fields {
		existing[v.Name] = v
	}
	var missing []*Field
	for _, v := range newTable.Fields {
		prev := existing[v.Name]
		if prev == nil {
			// Check if we can add the field
			if v.Constraint(ConstraintNotNull) != nil && !fieldHasDefault(m, v) {
				return fmt.Errorf("can't add NOT NULL field %q to table %q without a default value", v.Name, m.Table())
			}
			if v.Constraint(ConstraintPrimaryKey) != nil {
				return fmt.Errorf("can't add PRIMARY KEY field %q to table %q", v.Name, m.Table())
			}
			missing = append(missing, v)
		} else {
			if prev.Type != v.Type {
				// Check the Kind
				k1, len1 := TypeKind(prev.Type)
				k2, len2 := TypeKind(v.Type)
				if k1 == k2 {
					// Check lengths
					if len1 != len2 {
					}
					continue
				}
				// Check if we can transform the kind
				fields := m.Fields()
				idx := fields.MNameMap[v.Name]
				modelName := fields.QNames[idx]
				modelType := fields.Types[idx]
				return fmt.Errorf("field %q on table %q is of type %s which is not compatible with the model field %q of type %s (%s)",
					v.Name, m.Table(), prev.Type, modelName, v.Type, modelType)
			}
		}
	}
	if len(missing) > 0 {
		if err := d.backend.AddFields(d.db, m, prevTable, newTable, missing); err != nil {
			return err
		}
	}
	return nil
}

func (d *Driver) where(buf *bytes.Buffer, m driver.Model, q query.Q, begin int) ([]interface{}, error) {
	var params []interface{}
	var err error
	if !isNil(q) {
		buf.WriteString(" WHERE ")
		err = d.condition(buf, &params, m, q, begin)
	}
	return params, err
}

func (d *Driver) condition(buf *bytes.Buffer, params *[]interface{}, m driver.Model, q query.Q, begin int) error {
	var err error
	switch x := q.(type) {
	case *query.Eq:
		if isNil(x.Value) {
			x.Value = nil
			err = d.clause(buf, params, m, "%s IS NULL", &x.Field, begin)
		} else {
			err = d.clause(buf, params, m, "%s = %s", &x.Field, begin)
		}
	case *query.Neq:
		if isNil(x.Value) {
			x.Value = nil
			err = d.clause(buf, params, m, "%s IS NOT NULL", &x.Field, begin)
		} else {
			err = d.clause(buf, params, m, "%s != %s", &x.Field, begin)
		}
	case *query.Lt:
		err = d.clause(buf, params, m, "%s < %s", &x.Field, begin)
	case *query.Lte:
		err = d.clause(buf, params, m, "%s <= %s", &x.Field, begin)
	case *query.Gt:
		err = d.clause(buf, params, m, "%s > %s", &x.Field, begin)
	case *query.Gte:
		err = d.clause(buf, params, m, "%s >= %s", &x.Field, begin)
	case *query.In:
		value := reflect.ValueOf(x.Value)
		if value.Type().Kind() != reflect.Slice {
			return fmt.Errorf("argument for IN must be a slice (field %s)", x.Field.Field)
		}
		dbName, _, err := m.Map(x.Field.Field)
		if err != nil {
			return err
		}
		vLen := value.Len()
		if vLen == 0 {
			return fmt.Errorf("empty IN (field %s)", x.Field.Field)
		}
		buf.WriteString(dbName)
		buf.WriteString(" IN (")
		jj := len(*params) + begin + 1
		for ii := 0; ii < vLen; ii++ {
			*params = append(*params, value.Index(ii).Interface())
			buf.WriteString(d.backend.Placeholder(jj))
			buf.WriteByte(',')
			jj++
		}
		buf.Truncate(buf.Len() - 1)
		buf.WriteByte(')')
	case *query.And:
		err = d.conditions(buf, params, m, x.Conditions, " AND ", begin)
	case *query.Or:
		err = d.conditions(buf, params, m, x.Conditions, " OR ", begin)
	}
	return err
}

func (d *Driver) clause(buf *bytes.Buffer, params *[]interface{}, m driver.Model, format string, f *query.Field, begin int) error {
	dbName, _, err := m.Map(f.Field)
	if err != nil {
		return err
	}
	if f.Value != nil {
		if field, ok := f.Value.(query.F); ok {
			fName, _, err := m.Map(string(field))
			if err != nil {
				return err
			}
			fmt.Fprintf(buf, format, dbName, fName)
			return nil
		}
		*params = append(*params, f.Value)
		fmt.Fprintf(buf, format, dbName, d.backend.Placeholder(len(*params)+begin))
		return nil
	}
	fmt.Fprintf(buf, format, dbName)
	return nil
}

func (d *Driver) conditions(buf *bytes.Buffer, params *[]interface{}, m driver.Model, q []query.Q, sep string, begin int) error {
	buf.WriteByte('(')
	for _, v := range q {
		err := d.condition(buf, params, m, v, begin)
		if err != nil {
			return err
		}
		buf.WriteString(sep)
	}
	buf.Truncate(buf.Len() - len(sep))
	buf.WriteByte(')')
	return nil
}

func (d *Driver) SelectStmt(buf *bytes.Buffer, params *[]interface{}, fields []string, quote bool, m driver.Model) error {
	buf.WriteString("SELECT ")
	if fields != nil {
		if quote {
			for _, v := range fields {
				buf.WriteByte('"')
				buf.WriteString(v)
				buf.WriteByte('"')
				buf.WriteByte(',')
			}
		} else {
			for _, v := range fields {
				buf.WriteString(v)
				buf.WriteByte(',')
			}
		}
	} else {
		// Select all fields for the given model (which might be joined)
		cur := m
		for {
			if !cur.Skip() {
				for _, v := range cur.Fields().QuotedNames {
					buf.WriteString(v)
					buf.WriteByte(',')
				}
			}
			join := cur.Join()
			if join == nil {
				break
			}
			cur = join.Model()
		}
	}
	buf.Truncate(buf.Len() - 1)
	buf.WriteString(" FROM ")
	buf.WriteByte('"')
	buf.WriteString(m.Table())
	buf.WriteByte('"')
	for join := m.Join(); join != nil; {
		jm := join.Model()
		switch join.Type() {
		case driver.OuterJoin:
			buf.WriteString(" FULL OUTER")
		case driver.LeftJoin:
			buf.WriteString(" LEFT OUTER")
		case driver.RightJoin:
			buf.WriteString(" RIGHT OUTER")
		}
		buf.WriteString(" JOIN ")
		buf.WriteByte('"')
		buf.WriteString(jm.Table())
		buf.WriteByte('"')
		buf.WriteString(" ON ")
		if err := d.condition(buf, params, m, join.Query(), len(*params)); err != nil {
			return err
		}
		join = jm.Join()
	}
	return nil
}

func (d *Driver) Select(fields []string, quote bool, m driver.Model, q query.Q, sort []driver.Sort, limit int, offset int) (*bytes.Buffer, []interface{}, error) {
	buf := getBuffer()
	var params []interface{}
	if err := d.SelectStmt(buf, &params, fields, quote, m); err != nil {
		return nil, nil, err
	}
	qParams, err := d.where(buf, m, q, 0)
	if err != nil {
		return nil, nil, err
	}
	params = append(params, qParams...)
	if len(sort) > 0 {
		buf.WriteString(" ORDER BY ")
		for _, v := range sort {
			dbName, _, err := m.Map(v.Field())
			if err != nil {
				return nil, nil, err
			}
			buf.WriteString(dbName)
			if v.Direction() == driver.DESC {
				buf.WriteString(" DESC")
			}
			buf.WriteByte(',')
		}
		buf.Truncate(buf.Len() - 1)
	}
	if limit >= 0 {
		buf.WriteString(" LIMIT ")
		buf.WriteString(strconv.Itoa(limit))
	}
	if offset >= 0 {
		buf.WriteString(" OFFSET ")
		buf.WriteString(strconv.Itoa(offset))
	}
	return buf, params, nil
}

func (d *Driver) Begin() (driver.Tx, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	drv := *d
	drv.db = tx
	tx.driver = &drv
	return &drv, nil
}

func (d *Driver) Commit() error {
	return d.db.Commit()
}

func (d *Driver) Rollback() error {
	return d.db.Rollback()
}

func (d *Driver) Transaction(f func(driver.Driver) error) error {
	return nil
}

func (d *Driver) Capabilities() driver.Capability {
	return driver.CAP_JOIN | driver.CAP_TRANSACTION | driver.CAP_BEGIN |
		driver.CAP_AUTO_ID | driver.CAP_AUTO_INCREMENT | driver.CAP_PK |
		driver.CAP_COMPOSITE_PK | driver.CAP_UNIQUE | driver.CAP_DEFAULTS |
		d.backend.Capabilities()
}

func (d *Driver) HasFunc(fname string, retType reflect.Type) bool {
	fn, err := d.backend.Func(fname, retType)
	return err == nil && fn != ""
}

func (d *Driver) Connection() interface{} {
	return d.db
}

func NewDriver(b Backend, url *config.URL) (*Driver, error) {
	conn, err := sql.Open(b.Name(), url.ValueAndQuery())
	if err != nil {
		return nil, err
	}
	if mc, ok := url.Fragment.Int("max_conns"); ok {
		setMaxConns(conn, mc)
	}
	if mic, ok := url.Fragment.Int("max_idle_conns"); ok {
		conn.SetMaxIdleConns(mic)
	}
	var transforms map[reflect.Type]struct{}
	if tt := b.Transforms(); len(tt) > 0 {
		transforms = make(map[reflect.Type]struct{}, len(tt)*2)
		for _, v := range tt {
			transforms[v] = struct{}{}
			transforms[v.Elem()] = struct{}{}
		}
	}
	driver := &Driver{backend: b, transforms: transforms}
	driver.db = &DB{sqlDb: conn, conn: conn, driver: driver, replacesPlaceholders: b.Placeholder(0) != "?"}
	return driver, nil
}

// Assume s is quoted
func unquote(s string) string {
	p := strings.Index(s, ".")
	return s[p+2 : len(s)-1]
}

func fieldHasDefault(m driver.Model, f *Field) bool {
	if f.Default != "" {
		return true
	}
	fields := m.Fields()
	idx := fields.MNameMap[f.Name]
	return fields.HasDefault(idx)
}
