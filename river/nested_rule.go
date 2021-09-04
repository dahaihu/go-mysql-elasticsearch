package river

import (
	"fmt"

	"github.com/siddontang/go-log/log"
	"github.com/siddontang/go-mysql-elasticsearch/elastic"
	"github.com/siddontang/go-mysql/canal"
)

type NestedRule struct {
	Rule

	NestedField string `toml:"nested_filed"`
	// used to add,update, del nested field
	NestedPrimaryKey string `toml:"nested_primary_key"`
}

func (r *NestedRule) makeRequest(action string, rows [][]interface{}) ([]*elastic.BulkRequest, error) {
	reqs := make([]*elastic.BulkRequest, 0, len(rows))

	for _, values := range rows {
		req := &elastic.BulkRequest{
			Index:         r.getIndex(values),
			Type:          r.Type,
			ID:            r.getDocID(values),
			NestedRequest: true,
			Pipeline:      r.Pipeline,
			Action:        elastic.ActionUpdate,
		}
		data := make(map[string]interface{}, len(values))
		for idx, column := range r.TableInfo.Columns {
			var syn bool
			for _, filter := range r.Filter {
				if filter == column.Name {
					syn = true
					break
				}
			}
			if syn {
				data[column.Name] = values[idx]
			}
		}
		switch action {
		case canal.DeleteAction:
			req.Data = r.makeDeleteData(data)
		case canal.UpdateAction, canal.InsertAction:
			req.Data = r.makeUpdateData(data)
		//case canal.InsertAction:
		//	req.Data = r.makeInsertData(data)
		default:
			log.Errorf("invalid canal action %s", action)
			continue
		}
		esInsertNum.WithLabelValues(r.Index).Inc()
		//log.Infof("action is %s, index is %s, type is %s, data is %v\n",
		//	req.Action, req.Index, req.Type, req.Data,
		//)
		reqs = append(reqs, req)
	}
	return reqs, nil
}

func (r *NestedRule) makeUpdateData(data map[string]interface{}) map[string]interface{} {
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

// make delete nested filed data
func (r *NestedRule) makeDeleteData(data map[string]interface{}) map[string]interface{} {
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
func (r *NestedRule) makeInsertData(data map[string]interface{}) map[string]interface{} {
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
