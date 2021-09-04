package river

import (
	"github.com/siddontang/go-mysql-elasticsearch/elastic"
	"github.com/siddontang/go-mysql/schema"
)

type RuleInterface interface {
	makeRequest(string, [][]interface{}) ([]*elastic.BulkRequest, error)
	getTable() string
	getSchema() string
	setTableInfo(*schema.Table)
	getTableInfo() *schema.Table
}
