package sql

import (
	"bytes"
	"database/sql"
	"fmt"
	"gondola/log"
	"gondola/orm/driver"
	"gondola/orm/query"
	"reflect"
	"strconv"
	"strings"
)

type Driver struct {
	db         *db
	logger     *log.Logger
	backend    Backend
	transforms map[reflect.Type]struct{}
}

func (d *Driver) MakeModels(ms []driver.Model) error {
	// Create tables
	// TODO: References
	for _, v := range ms {
		tableFields, err := d.tableFields(v)
		if err != nil {
			return err
		}
		if len(tableFields) == 0 {
			log.Debugf("Skipping collection %s (model %v) because it has no fields", v.Collection, v)
			continue
		}
		sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n);", v.Collection(), strings.Join(tableFields, ",\n"))
		d.debugq(sql, nil)
		_, err = d.db.Exec(sql)
		if err != nil {
			return err
		}
	}
	// Create indexes
	for _, v := range ms {
		for _, idx := range v.Indexes() {
			name, err := d.indexName(v, idx)
			if err != nil {
				return err
			}
			err = d.backend.Index(d.db, v, idx, name)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *Driver) Query(m driver.Model, q query.Q, limit int, offset int, sort int, sortField string) driver.Iter {
	query, params, err := d.Select(m.Fields().Names, m, q, limit, offset, sort, sortField)
	if err != nil {
		return &Iter{err: err}
	}
	d.debugq(query, params)
	rows, err := d.db.Query(query, params...)
	if err != nil {
		return &Iter{err: err}
	}
	return &Iter{model: m, rows: rows, driver: d}
}

func (d *Driver) Count(m driver.Model, q query.Q, limit int, offset int, sort int, sortField string) (uint64, error) {
	var count uint64
	query, params, err := d.Select([]string{"COUNT(*)"}, m, q, limit, offset, sort, sortField)
	if err != nil {
		return 0, err
	}
	d.debugq(query, params)
	err = d.db.QueryRow(query, params...).Scan(&count)
	return count, err
}

func (d *Driver) Insert(m driver.Model, data interface{}) (driver.Result, error) {
	_, fields, values, err := d.saveParameters(m, data)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO ")
	buf.WriteString(m.Collection())
	buf.WriteString(" (")
	for _, v := range fields {
		buf.WriteString(v)
		buf.WriteByte(',')
	}
	buf.Truncate(buf.Len() - 1)
	buf.WriteString(") VALUES (")
	buf.WriteString(d.backend.Placeholders(len(fields)))
	buf.WriteByte(')')
	return d.backend.Insert(d.db, m, buf.String(), values...)
}

func (d *Driver) Update(m driver.Model, data interface{}, q query.Q) (driver.Result, error) {
	_, fields, values, err := d.saveParameters(m, data)
	if err != nil {
		return nil, err
	}
	where, qParams, err := d.where(m, q, len(values))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteString("UPDATE ")
	buf.WriteString(m.Collection())
	buf.WriteString(" SET ")
	for ii, v := range fields {
		buf.WriteString(v)
		buf.WriteByte('=')
		buf.WriteString(d.backend.Placeholder(ii + 1))
		buf.WriteByte(',')
	}
	// remove last ,
	buf.Truncate(buf.Len() - 1)
	if where != "" {
		buf.WriteString(" WHERE ")
		buf.WriteString(where)
	}
	params := append(values, qParams...)
	query := buf.String()
	d.debugq(query, params)
	return d.db.Exec(query, params...)
}

func (d *Driver) Upsert(m driver.Model, data interface{}, q query.Q) (driver.Result, error) {
	// TODO: MySql might be able to provide upserts
	return nil, nil
}

func (d *Driver) Delete(m driver.Model, q query.Q) (driver.Result, error) {
	where, params, err := d.where(m, q, 0)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteString("DELETE FROM ")
	buf.WriteString(m.Collection())
	if where != "" {
		buf.WriteString(" WHERE ")
		buf.WriteString(where)
	}
	query := buf.String()
	d.debugq(query, params)
	return d.db.Exec(query, params...)
}

func (d *Driver) Close() error {
	return d.db.Close()
}

func (d *Driver) Upserts() bool {
	return false
}

func (d *Driver) Tags() []string {
	return []string{d.backend.Tag(), "sql"}
}

func (d *Driver) DB() *sql.DB {
	return d.db.DB
}

func (d *Driver) DBBackend() Backend {
	return d.backend
}

func (d *Driver) SetLogger(logger *log.Logger) {
	d.logger = logger
}

func (d *Driver) debugq(sql string, args interface{}) {
	if d.logger != nil {
		d.logger.Debugf("SQL %q with arguments %v", sql, args)
	}
}

func (d *Driver) saveParameters(m driver.Model, data interface{}) (reflect.Value, []string, []interface{}, error) {
	// data is guaranteed to be of m.Type()
	val := reflect.ValueOf(data)
	for val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	fields := m.Fields()
	max := len(fields.Names)
	names := make([]string, 0, max)
	values := make([]interface{}, 0, max)
	var err error
	if d.transforms != nil {
		for ii, v := range fields.Indexes {
			f := val.FieldByIndex(v)
			if fields.OmitZero[ii] && driver.IsZero(f) {
				continue
			}
			var fval interface{}
			if _, ok := d.transforms[f.Type()]; ok {
				fval, err = d.backend.TransformOutValue(f)
				if err != nil {
					return val, nil, nil, err
				}
			} else if !fields.NullZero[ii] || !driver.IsZero(f) {
				if fields.Tags[ii].Has("json") {
					fval, err = encodeJson(f)
					if err != nil {
						return val, nil, nil, err
					}
				} else {
					fval = f.Interface()
				}
			}
			names = append(names, fields.Names[ii])
			values = append(values, fval)
		}
	} else {
		for ii, v := range fields.Indexes {
			f := val.FieldByIndex(v)
			if fields.OmitZero[ii] && driver.IsZero(f) {
				continue
			}
			var fval interface{}
			if !fields.NullZero[ii] || !driver.IsZero(f) {
				if fields.Tags[ii].Has("json") {
					fval, err = encodeJson(f)
					if err != nil {
						return val, nil, nil, err
					}
				} else {
					fval = f.Interface()
				}
			}
			names = append(names, fields.Names[ii])
			values = append(values, fval)
		}
	}
	return val, names, values, nil
}

func (d *Driver) outValues(m driver.Model, out interface{}) ([]interface{}, []scanner, error) {
	val := reflect.ValueOf(out)
	vt := val.Type()
	if vt.Kind() != reflect.Ptr {
		return nil, nil, fmt.Errorf("can't set object of type %t. Please, pass a %v rather than a %v", out, reflect.PtrTo(vt), vt)
	}
	if vt.Elem().Kind() == reflect.Ptr && vt.Elem().Elem().Kind() == reflect.Struct {
		// Received a pointer to pointer
		val = val.Elem()
		el := reflect.New(val.Type().Elem())
		val.Set(el)
	}
	for val.Type().Kind() == reflect.Ptr {
		el := val.Elem()
		if !el.IsValid() {
			el = reflect.New(val.Type().Elem())
			val.Set(el)
		}
		val = el
	}
	fields := m.Fields()
	values := make([]interface{}, len(fields.Indexes))
	scanners := make([]scanner, len(fields.Indexes))
	for ii, v := range fields.Indexes {
		field := val.FieldByIndex(v)
		tag := fields.Tags[ii]
		var s scanner
		if _, ok := d.transforms[field.Type()]; ok {
			s = BackendScanner(&field, tag, d.backend)
		} else {
			s = Scanner(&field, tag)
		}
		scanners[ii] = s
		values[ii] = s
	}
	return values, scanners, nil
}

func (d *Driver) tableFields(m driver.Model) ([]string, error) {
	fields := m.Fields()
	names := fields.Names
	types := fields.Types
	tags := fields.Tags
	dbFields := make([]string, len(names))
	for ii, v := range names {
		typ := types[ii]
		tag := tags[ii]
		// Check json encoded types
		if tag.Has("json") {
			if err := tryEncodeJson(typ, d); err != nil {
				return nil, fmt.Errorf("can't encode field %q as JSON: %s", fields.QNames[ii], err)
			}
		}
		ft, err := d.backend.FieldType(typ, tag)
		if err != nil {
			return nil, err
		}
		opts, err := d.backend.FieldOptions(typ, tag)
		if err != nil {
			return nil, err
		}
		dbFields[ii] = fmt.Sprintf("%s %s %s", v, ft, strings.Join(opts, " "))
	}
	return dbFields, nil
}

func (d *Driver) where(m driver.Model, q query.Q, begin int) (string, []interface{}, error) {
	var params []interface{}
	clause, err := d.condition(m.Fields(), q, &params, begin)
	return clause, params, err
}

func (d *Driver) condition(fields *driver.Fields, q query.Q, params *[]interface{}, begin int) (string, error) {
	var clause string
	var err error
	switch x := q.(type) {
	case *query.Eq:
		if isNil(x.Value) {
			x.Value = nil
			clause, err = d.clause(fields, "%s IS NULL", &x.Field, params, begin)
		} else {
			clause, err = d.clause(fields, "%s = %s", &x.Field, params, begin)
		}
	case *query.Neq:
		if isNil(x.Value) {
			x.Value = nil
			clause, err = d.clause(fields, "%s IS NOT NULL", &x.Field, params, begin)
		} else {
			clause, err = d.clause(fields, "%s != %s", &x.Field, params, begin)
		}
	case *query.Lt:
		clause, err = d.clause(fields, "%s < %s", &x.Field, params, begin)
	case *query.Lte:
		clause, err = d.clause(fields, "%s <= %s", &x.Field, params, begin)
	case *query.Gt:
		clause, err = d.clause(fields, "%s > %s", &x.Field, params, begin)
	case *query.Gte:
		clause, err = d.clause(fields, "%s >= %s", &x.Field, params, begin)
	case *query.In:
		value := reflect.ValueOf(x.Value)
		if value.Type().Kind() != reflect.Slice {
			return "", fmt.Errorf("argument for IN must be a slice (field %s)", x.Field.Field)
		}
		dbName, _, err := fields.Map(x.Field.Field)
		if err != nil {
			return "", err
		}
		vLen := value.Len()
		placeholders := make([]string, vLen)
		jj := len(*params) + begin + 1
		for ii := 0; ii < vLen; ii++ {
			*params = append(*params, value.Index(ii).Interface())
			placeholders[ii] = d.backend.Placeholder(jj)
			jj++
		}
		clause = fmt.Sprintf("%s IN (%s)", dbName, strings.Join(placeholders, ","))
	case *query.And:
		clause, err = d.conditions(fields, x.Conditions, " AND ", params, begin)
	case *query.Or:
		clause, err = d.conditions(fields, x.Conditions, " OR ", params, begin)
	}
	return clause, err
}

func (d *Driver) clause(fields *driver.Fields, format string, f *query.Field, params *[]interface{}, begin int) (string, error) {
	dbName, _, err := fields.Map(f.Field)
	if err != nil {
		return "", err
	}
	if f.Value != nil {
		*params = append(*params, f.Value)
		return fmt.Sprintf(format, dbName, d.backend.Placeholder(len(*params)+begin)), nil
	}
	return fmt.Sprintf(format, dbName), nil
}

func (d *Driver) conditions(fields *driver.Fields, q []query.Q, sep string, params *[]interface{}, begin int) (string, error) {
	clauses := make([]string, len(q))
	for ii, v := range q {
		c, err := d.condition(fields, v, params, begin)
		if err != nil {
			return "", err
		}
		clauses[ii] = c
	}
	return fmt.Sprintf("(%s)", strings.Join(clauses, sep)), nil
}

func (d *Driver) indexName(m driver.Model, idx driver.Index) (string, error) {
	indexFields := idx.Fields()
	if len(indexFields) == 0 {
		return "", fmt.Errorf("index on %v has no fields", m.Type())
	}
	var buf bytes.Buffer
	buf.WriteString(m.Collection())
	fields := m.Fields()
	for _, v := range indexFields {
		dbName, _, err := fields.Map(v)
		if err != nil {
			return "", err
		}
		buf.WriteByte('_')
		buf.WriteString(dbName)
	}
	return buf.String(), nil
}

func (d *Driver) Select(fields []string, m driver.Model, q query.Q, limit int, offset int, sort int, sortField string) (string, []interface{}, error) {
	where, params, err := d.where(m, q, 0)
	if err != nil {
		return "", nil, err
	}
	var buf bytes.Buffer
	buf.WriteString("SELECT ")
	for _, v := range fields {
		buf.WriteString(v)
		buf.WriteByte(',')
	}
	buf.Truncate(buf.Len() - 1)
	buf.WriteString(" FROM ")
	buf.WriteString(m.Collection())
	if where != "" {
		buf.WriteString(" WHERE ")
		buf.WriteString(where)
	}
	if sort != driver.NONE {
		buf.WriteString(" ORDER BY ")
		buf.WriteString(sortField)
		switch sort {
		case driver.ASC:
			buf.WriteString(" ASC")
		case driver.DESC:
			buf.WriteString(" DESC")
		}
	}
	if limit >= 0 {
		buf.WriteString(" LIMIT ")
		buf.WriteString(strconv.Itoa(limit))
	}
	if offset >= 0 {
		buf.WriteString(" OFFSET ")
		buf.WriteString(strconv.Itoa(offset))
	}
	return buf.String(), params, nil
}

func NewDriver(b Backend, params string) (*Driver, error) {
	conn, err := sql.Open(b.Name(), params)
	if err != nil {
		return nil, err
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
	driver.db = &db{DB: conn, driver: driver}
	return driver, nil
}
