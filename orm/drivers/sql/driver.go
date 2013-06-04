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

// Backend is the interface implemented by drivers
// for database/sql orm backends
type Backend interface {
	Name() string
	FieldType(reflect.Type, driver.Tag) (string, error)
	FieldOptions(reflect.Type, driver.Tag) ([]string, error)
}

type Driver struct {
	db      *sql.DB
	backend Backend
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

func (d *Driver) Query(m driver.Model, q query.Q) driver.Iter {
	where, params := d.where(q)
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s", strings.Join(m.FieldNames(), ","), m.Collection(), where)
	rows, err := d.db.Query(query, params...)
	if err != nil {
		return &Iter{err: err}
	}
	return &Iter{model: m, rows: rows}
}

func (d *Driver) Insert(m driver.Model, data interface{}) (driver.Result, error) {
	value, fields, values, err := m.Insert(data)
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
	return nil, nil
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

func (d *Driver) where(q query.Q) (string, []interface{}) {
	var params []interface{}
	clause := d.condition(q, params)
	return clause, params
}

func (d *Driver) condition(q query.Q, params []interface{}) string {
	var clause string
	switch x := q.(type) {
	case *query.Eq:
		clause = fmt.Sprintf("%s = ?", x.Field)
		params = append(params, x.Value)
	case *query.Neq:
		clause = fmt.Sprintf("%s != ?", x.Field)
		params = append(params, x.Value)
	case *query.Lt:
		clause = fmt.Sprintf("%s < ?", x.Field)
		params = append(params, x.Value)
	case *query.Lte:
		clause = fmt.Sprintf("%s <= ?", x.Field)
		params = append(params, x.Value)
	case *query.Gt:
		clause = fmt.Sprintf("%s > ?", x.Field)
		params = append(params, x.Value)
	case *query.Gte:
		clause = fmt.Sprintf("%s >= ?", x.Field)
		params = append(params, x.Value)
	case *query.And:
		clause = d.conditions(x.Conditions, " AND ", params)
	case *query.Or:
		clause = d.conditions(x.Conditions, " OR ", params)
	}
	if clause == "" {
		clause = "1"
	}
	return clause
}

func (d *Driver) conditions(q []query.Q, sep string, params []interface{}) string {
	clauses := make([]string, len(q))
	for ii, v := range q {
		clauses[ii] = d.condition(v, params)
	}
	return fmt.Sprintf("(%s)", strings.Join(clauses, sep))
}

func NewDriver(b Backend, params string) (*Driver, error) {
	db, err := sql.Open(b.Name(), params)
	if err != nil {
		return nil, err
	}
	return &Driver{db: db, backend: b}, nil
}
