package river

import (
	"fmt"

	"github.com/siddontang/go-mysql-elasticsearch/elastic"
)

type NestedRule struct {
	Rule

	NestedField string `toml:"nested_filed"`
	// used to add,update, del nested field
	NestedPrimaryKey string `toml:"nested_primary_key"`
}

func (r *NestedRule) makeInsertReqData(req *elastic.BulkRequest, values []interface{}) {
	req.Data = make(map[string]interface{}, len(values))
	req.NestedRequest = true
	req.Action = elastic.ActionUpdate
	data := make(map[string]interface{}, len(values))
	for idx, column := range r.TableInfo.Columns {
		data[column.Name] = values[idx]
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
			"params": data,
		},
	}
	return
}

// make delete nested filed data
func (r *NestedRule) makeNestedFieldDelRequest(data map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"script": map[string]interface{}{
			"source": fmt.Sprintf(
				`ctx._source.%s.removeIf(item -> item.%s == params.%s)`,
				r.NestedField, r.NestedPrimaryKey, r.NestedPrimaryKey,
			),
			"params": data,
		},
	}
}

// make insert nested filed data
func (r *NestedRule)makeNestedFieldInsertRequest(data map[string]interface{}, ) map[string]interface{} {
	return map[string]interface{}{
		"script": map[string]interface{}{
			"source": fmt.Sprintf(`
					if (ctx._source.%[1]s == null) {
							ctx._source.%[1]s = new ArrayList();
						}
					ctx._source.%[1]s.add(params);`,
				r.NestedPrimaryKey,
			),
			"params": data,
		},
	}
}

// make nested field update data
func (r *NestedRule)makeNestedFieldUpdateRequest(data map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"script": map[string]interface{}{
			"source": fmt.Sprintf(
				`if (ctx._source.%[1]s == null) {
							ctx._source.%[1]s = new ArrayList();
						}
						ctx._source.%[1]s.removeIf(item -> item.%[2]s == params.%[2]s); 
						ctx._source.%[1]s.add(params)`,
				r.NestedField, r.NestedPrimaryKey,
			),
			"params": data,
		},
	}
}
