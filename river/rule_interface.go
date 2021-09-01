package river

import "github.com/siddontang/go-mysql-elasticsearch/elastic"

type RuleInterface interface {
	makeInsertReqData(*elastic.BulkRequest, []interface{})
}
