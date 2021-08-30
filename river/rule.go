package river

import (
	"github.com/siddontang/go-mysql-elasticsearch/elastic"
	"strings"

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
	Index       string `toml:"index"`
	IndexField  string `toml:"index_field"`
	NestedRule  bool   `toml:"nested_rule"`
	NestedField string `toml:"nested_filed"`
	// nested field primary key
	NestedPrimaryKey string   `toml:"nested_primary_key"`
	Type             string   `toml:"type"`
	Parent           string   `toml:"parent"`
	ID               []string `toml:"id"`

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

func (r *Rule) makeInsertReqData(req *elastic.BulkRequest, river *River,
	values []interface{}) {
	req.Data = make(map[string]interface{}, len(values))
	req.Action = elastic.ActionIndex
	for i, c := range r.TableInfo.Columns {
		if !r.CheckFilter(c.Name) {
			continue
		}
		mapped := false
		for k, v := range r.FieldMapping {
			mysql, elastic, fieldType := river.getFieldParts(k, v)
			if mysql == c.Name {
				mapped = true
				req.Data[elastic] = river.getFieldValue(&c, fieldType, values[i])
			}
		}
		if mapped == false {
			req.Data[c.Name] = river.makeReqColumnData(&c, values[i])
		}
	}
}
