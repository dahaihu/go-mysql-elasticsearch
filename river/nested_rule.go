package river

import (
	"fmt"

	"github.com/siddontang/go-mysql-elasticsearch/elastic"

	"github.com/siddontang/go-mysql/schema"
)

type BaseRule struct {
	Schema     string `toml:"schema"`
	Table      string `toml:"table"`
	Index      string `toml:"index"`
	IndexField string `toml:"index_field"`
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
type NestedRule struct {
	BaseRule

	Type string `toml:"type"`
	// the nested field in doc record
	NestedField string `toml:"nested_filed"`
	// used to add,update, del nested field
	NestedPrimaryKey string `toml:"nested_primary_key"`
}

func (r *NestedRule) makeInsertReqData(req *elastic.BulkRequest, river *River,
	values []interface{}) {
	req.Data = make(map[string]interface{}, len(values))
	req.NestedRequest = true
	req.Action = elastic.ActionUpdate
	allFields := make(map[string]interface{}, len(values))
	for idx, column := range r.TableInfo.Columns {
		allFields[column.Name] = values[idx]
		if column.Name == r.IndexField {
			req.ID = fmt.Sprintf("%v", values[idx])
		}
	}
	// use the script to add data
	req.Data = map[string]interface{}{
		"script": map[string]interface{}{
			"source": fmt.Sprintf(
			`if (ctx._source.%[1]s == null) {
						ctx._source.%[1]s = new ArrayList();
					}
					ctx._source.%[1]s.removeIf(item -> item.%[2]s == params.%[2]s); 
					ctx._source.%[1]s.add(params)`,
				r.NestedField, r.NestedPrimaryKey,
			),
			"params": allFields,
		},
	}
	return
}
