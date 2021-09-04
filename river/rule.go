package river

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/siddontang/go-mysql-elasticsearch/elastic"

	"github.com/siddontang/go-log/log"
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/schema"
)

// Rule is the rule for how to sync data from MySQL to ES.
// If you want to sync MySQL data into elasticsearch, you must set a rule to let use know how to do it.
// The mapping rule may thi: schema + table <-> index + document type.
// schema and table is for MySQL, index and document type is for Elasticsearch.
type Rule struct {
	Schema string `toml:"schema"`
	Table  string `toml:"table"`
	// only one could be used
	Index      string   `toml:"index"`
	IndexField string   `toml:"index_field"`
	Type       string   `toml:"type"`
	Parent     string   `toml:"parent"`
	ID         []string `toml:"id"`

	// Default, a MySQL table field name is mapped to Elasticsearch field name.
	// Sometimes, you want to use different name, e.g, the MySQL file name is title,
	// but in Elasticsearch, you want to name it my_title.
	FieldMapping map[string]string `toml:"field"`

	// MySQL table information
	TableInfo *schema.Table

	//only MySQL fields in filter will be synced , default sync all fields
	Filter []string `toml:"filter"`

	// Elasticsearch pipeline
	// To pre-process documents before indexing
	Pipeline string `toml:"pipeline"`
}

func (r *Rule) getIndex(rows []interface{}) string {
	// todo check index field exist when read config
	if len(r.Index) != 0 {
		return r.Index
	}
	for idx, column := range r.TableInfo.Columns {
		if column.Name == r.IndexField {
			return fmt.Sprintf("%v", rows[idx])
		}
	}
	panic("invalid rows info")
}

func (r *Rule) getDocID(row []interface{}) string {
	// todo error process
	var ids []interface{}
	if r.ID == nil {
		ids, _ = r.TableInfo.GetPKValues(row)
	} else {
		ids = make([]interface{}, 0, len(r.ID))
		for _, column := range r.ID {
			value, _ := r.TableInfo.GetColumnValue(column, row)
			ids = append(ids, value)
		}
	}

	var buf bytes.Buffer

	sep := ""
	for _, value := range ids {
		buf.WriteString(fmt.Sprintf("%s%v", sep, value))
		sep = ":"
	}

	return buf.String()
}

func newDefaultRule(schema string, table string) *Rule {
	r := new(Rule)

	r.Schema = schema
	r.Table = table

	lowerTable := strings.ToLower(table)
	r.Index = lowerTable
	r.Type = lowerTable

	r.FieldMapping = make(map[string]string)

	return r
}

func (r *Rule) prepare() error {
	if r.FieldMapping == nil {
		r.FieldMapping = make(map[string]string)
	}

	if len(r.Index) == 0 {
		r.Index = r.Table
	}

	if len(r.Type) == 0 {
		r.Type = r.Index
	}

	// ES must use a lower-case Type
	// Here we also use for Index
	r.Index = strings.ToLower(r.Index)
	r.Type = strings.ToLower(r.Type)

	return nil
}

// CheckFilter checkers whether the field needs to be filtered.
func (r *Rule) CheckFilter(field string) bool {
	if r.Filter == nil {
		return true
	}

	for _, f := range r.Filter {
		if f == field {
			return true
		}
	}
	return false
}

func (r *Rule) makeRequest(action string, rows [][]interface{}) ([]*elastic.BulkRequest, error) {
	log.Printf("schame is %s, table is %s\n", r.Schema, r.Table)
	reqs := make([]*elastic.BulkRequest, 0, len(rows))
	var elasticAction string
	switch action {
	case canal.InsertAction:
		elasticAction = elastic.ActionIndex
	case canal.UpdateAction:
		elasticAction = elastic.ActionUpdate
	case canal.DeleteAction:
		elasticAction = elastic.ActionDelete
	}
	for _, values := range rows {
		req := &elastic.BulkRequest{
			Index:         r.getIndex(values),
			Type:          r.Type,
			ID:            r.getDocID(values),
			NestedRequest: false,
			Pipeline:      r.Pipeline,
			Action:        elasticAction,
		}
		req.Data = make(map[string]interface{}, len(values))
		for idx, column := range r.TableInfo.Columns {
			req.Data[column.Name] = values[idx]
		}
		//log.Infof("action is %s, index is %s, type is %s, data is %v\n",
		//	req.Action, req.Index, req.Type, req.Data,
		//)
		reqs = append(reqs, req)
	}

	return reqs, nil
}

func (r *Rule) setTableInfo(tableInfo *schema.Table) {
	r.TableInfo = tableInfo
}

func (r *Rule) getTableInfo() *schema.Table {
	return r.TableInfo
}

func (r *Rule) getTable() string {
	return r.Table
}

func (r *Rule) getSchema() string {
	return r.Schema
}
