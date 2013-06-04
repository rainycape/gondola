package sql

import (
	"database/sql"
	"fmt"
	"gondola/log"
	"gondola/orm/driver"
	"gondola/orm/query"
	"reflect"
	"strings"
)

const placeholders = "?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?"

type Driver struct {
	db         *sql.DB
	backend    Backend
	transforms map[reflect.Type]reflect.Type
}

func (d *Driver) MakeModels(ms []driver.Model) error {
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
		fmt.Println(sql)
		_, err = d.db.Exec(sql)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Driver) Query(m driver.Model, q query.Q, limit int, offset int) driver.Iter {
	where, params, err := d.where(m, q)
	if err != nil {
		return &Iter{err: err}
	}
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s", strings.Join(m.FieldNames(), ","), m.Collection(), where)
	if limit >= 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	if offset >= 0 {
		query += fmt.Sprintf(" OFFSET %d", offset)
	}
	fmt.Println(query, params)
	rows, err := d.db.Query(query, params...)
	if err != nil {
		return &Iter{err: err}
	}
	return &Iter{model: m, rows: rows, driver: d}
}

func (d *Driver) Insert(m driver.Model, data interface{}) (driver.Result, error) {
	value, fields, values, err := d.insertParameters(m, data)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", m.Collection(), strings.Join(fields, ","), d.placeholders(len(fields)))
	res, err := d.db.Exec(query, values...)
	if err == nil {
		fields := m.Fields()
		if pk := fields.PrimaryKey; pk >= 0 {
			id, err := res.LastInsertId()
			if err == nil && id != 0 {
				f := value.FieldByIndex(fields.Indexes[pk])
				if !f.CanSet() {
					t := value.Type()
					return nil, fmt.Errorf("can't set primary key field %q. Please, insert a %v rather than a %v",
						t.FieldByIndex(fields.Indexes[pk]).Name, reflect.PtrTo(t), t)
				}
				f.SetInt(id)
			}
		}
	}
	return res, err
}

func (d *Driver) Update(m driver.Model, data interface{}, q query.Q) (driver.Result, error) {
	return nil, nil
}

func (d *Driver) Delete(m driver.Model, q query.Q) (driver.Result, error) {
	where, params, err := d.where(m, q)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", m.Collection(), where)
	fmt.Println(query, params)
	return d.db.Exec(query, params...)
}

func (d *Driver) Close() error {
	return d.db.Close()
}

func (d *Driver) Tag() string {
	return "sql"
}

func (d *Driver) placeholders(n int) string {
	p := placeholders
	if n > 32 {
		p = strings.Repeat("?,", n)
	}
	return p[:2*n-1]
}

func (d *Driver) insertParameters(m driver.Model, data interface{}) (reflect.Value, []string, []interface{}, error) {
	// data is guaranteed to be of m.Type()
	val := reflect.ValueOf(data)
	for val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	fields := m.Fields()
	max := len(fields.Names)
	names := make([]string, 0, max)
	values := make([]interface{}, 0, max)
	if d.transforms != nil {
		for ii, v := range fields.Indexes {
			f := val.FieldByIndex(v)
			if fields.OmitZero[ii] && driver.IsZero(f) {
				continue
			}
			var fval interface{}
			if _, ok := d.transforms[f.Type()]; ok {
				var err error
				fval, err = d.backend.TransformOutValue(f)
				if err != nil {
					return val, nil, nil, err
				}
			} else if !fields.NullZero[ii] || !driver.IsZero(f) {
				fval = f.Interface()
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
				fval = f.Interface()
			}
			names = append(names, fields.Names[ii])
			values = append(values, fval)
		}
	}
	return val, names, values, nil
}

func (d *Driver) outValues(m driver.Model, out interface{}) ([]*transform, []interface{}, error) {
	var transforms []*transform
	val := reflect.ValueOf(out)
	vt := val.Type()
	if vt.Kind() != reflect.Ptr {
		return nil, nil, fmt.Errorf("can't set object of type %t. Please, pass a %v rather than a %v", out, reflect.PtrTo(val.Type()), val.Type())
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
	if d.transforms != nil {
		for ii, v := range fields.Indexes {
			field := val.FieldByIndex(v)
			if dbT, ok := d.transforms[field.Type()]; ok {
				dbVar := reflect.New(dbT)
				transforms = append(transforms, &transform{
					In:  dbVar.Elem(),
					Out: field,
				})
				values[ii] = dbVar.Interface()
			} else {
				values[ii] = field.Addr().Interface()
			}
		}
	} else {
		for ii, v := range fields.Indexes {
			values[ii] = val.FieldByIndex(v).Addr().Interface()
		}
	}
	return transforms, values, nil
}

func (d *Driver) tableFields(m driver.Model) ([]string, error) {
	names := m.FieldNames()
	fields := make([]string, len(names))
	for ii, v := range names {
		typ := m.FieldType(v)
		tag := m.FieldTag(v)
		ft, err := d.backend.FieldType(typ, tag)
		if err != nil {
			return nil, err
		}
		opts, err := d.backend.FieldOptions(typ, tag)
		if err != nil {
			return nil, err
		}
		fields[ii] = fmt.Sprintf("%s %s %s", v, ft, strings.Join(opts, " "))
	}
	return fields, nil
}

func (d *Driver) where(m driver.Model, q query.Q) (string, []interface{}, error) {
	var params []interface{}
	nmap := m.Fields().NameMap
	clause, err := d.condition(nmap, q, &params)
	return clause, params, err
}

func (d *Driver) condition(nmap map[string]string, q query.Q, params *[]interface{}) (string, error) {
	var clause string
	var err error
	switch x := q.(type) {
	case *query.Eq:
		clause, err = d.clause(nmap, "%s = ?", &x.Field, params)
	case *query.Neq:
		clause, err = d.clause(nmap, "%s != ?", &x.Field, params)
	case *query.Lt:
		clause, err = d.clause(nmap, "%s < ?", &x.Field, params)
	case *query.Lte:
		clause, err = d.clause(nmap, "%s <= ?", &x.Field, params)
	case *query.Gt:
		clause, err = d.clause(nmap, "%s > ?", &x.Field, params)
	case *query.Gte:
		clause, err = d.clause(nmap, "%s >= ?", &x.Field, params)
	case *query.And:
		clause, err = d.conditions(nmap, x.Conditions, " AND ", params)
	case *query.Or:
		clause, err = d.conditions(nmap, x.Conditions, " OR ", params)
	}
	if clause == "" {
		clause = "1"
	}
	return clause, err
}

func (d *Driver) clause(nmap map[string]string, format string, f *query.Field, params *[]interface{}) (string, error) {
	n := f.Field
	dbName, ok := nmap[n]
	if !ok {
		return "", fmt.Errorf("can't map field %q to database name", n)
	}
	*params = append(*params, f.Value)
	return fmt.Sprintf(format, dbName), nil
}

func (d *Driver) conditions(nmap map[string]string, q []query.Q, sep string, params *[]interface{}) (string, error) {
	clauses := make([]string, len(q))
	for ii, v := range q {
		c, err := d.condition(nmap, v, params)
		if err != nil {
			return "", err
		}
		clauses[ii] = c
	}
	return fmt.Sprintf("(%s)", strings.Join(clauses, sep)), nil
}

func NewDriver(b Backend, params string) (*Driver, error) {
	db, err := sql.Open(b.Name(), params)
	if err != nil {
		return nil, err
	}
	var transforms map[reflect.Type]reflect.Type
	if tt := b.Transforms(); len(tt) > 0 {
		transforms = make(map[reflect.Type]reflect.Type, len(tt)*2)
		for k, v := range tt {
			transforms[k] = v
			transforms[reflect.PtrTo(k)] = v
		}
	}
	return &Driver{db: db, backend: b, transforms: transforms}, nil
}
