package river

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/siddontang/go-log/log"
	"github.com/siddontang/go-mysql-elasticsearch/elastic"
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
	"github.com/siddontang/go-mysql/replication"
	"github.com/siddontang/go-mysql/schema"
)

const (
	fieldTypeList = "list"
	// for the mysql int type to es date type
	// set the [rule.field] created_time = ",date"
	fieldTypeDate = "date"
)

const mysqlDateFormat = "2006-01-02"

type posSaver struct {
	pos   mysql.Position
	force bool
}

type eventHandler struct {
	r *River
}

func (h *eventHandler) OnRotate(e *replication.RotateEvent) error {
	pos := mysql.Position{
		Name: string(e.NextLogName),
		Pos:  uint32(e.Position),
	}

	h.r.syncCh <- posSaver{pos, true}

	return h.r.ctx.Err()
}

func (h *eventHandler) OnTableChanged(schema, table string) error {
	log.Infof("%s.%s changed", schema, table)
	err := h.r.updateRule(schema, table)
	if err != nil && err != ErrRuleNotExist {
		return errors.Trace(err)
	}
	return nil
}

func (h *eventHandler) OnDDL(nextPos mysql.Position, _ *replication.QueryEvent) error {
	h.r.syncCh <- posSaver{nextPos, true}
	return h.r.ctx.Err()
}

func (h *eventHandler) OnXID(nextPos mysql.Position) error {
	h.r.syncCh <- posSaver{nextPos, false}
	return h.r.ctx.Err()
}

func (h *eventHandler) OnRow(e *canal.RowsEvent) error {
	//log.Info("action data is ", e.Action, e.Rows)
	rule, ok := h.r.rules[ruleKey(e.Table.Schema, e.Table.Name)]
	if !ok {
		return nil
	}
	//log.Infof("table info is %v", rule.getTableInfo())
	reqs, err := rule.makeRequest(e.Action, e.Rows)
	if err != nil {
		h.r.cancel()
		return errors.Errorf("make %s ES request err %v, close sync", e.Action, err)
	}

	h.r.syncCh <- reqs

	return h.r.ctx.Err()
}

func (h *eventHandler) OnGTID(gtid mysql.GTIDSet) error {
	return nil
}

func (h *eventHandler) OnPosSynced(pos mysql.Position, set mysql.GTIDSet, force bool) error {
	return nil
}

func (h *eventHandler) String() string {
	return "ESRiverEventHandler"
}

func (r *River) syncLoop() {
	bulkSize := r.c.BulkSize
	if bulkSize == 0 {
		bulkSize = 128
	}

	interval := r.c.FlushBulkTime.Duration
	if interval == 0 {
		interval = 200 * time.Millisecond
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer r.wg.Done()

	lastSavedTime := time.Now()
	reqs := make([]*elastic.BulkRequest, 0, 1024)

	var pos mysql.Position

	for {
		needFlush := false
		needSavePos := false
		// todo ticker update
		select {
		case v := <-r.syncCh:
			switch v := v.(type) {
			case posSaver:
				now := time.Now()
				if v.force || now.Sub(lastSavedTime) > 3*time.Second {
					lastSavedTime = now
					needFlush = true
					needSavePos = true
					pos = v.pos
				}
			case []*elastic.BulkRequest:
				reqs = append(reqs, v...)
				needFlush = len(reqs) >= bulkSize
			}
		case <-ticker.C:
			needFlush = true
		case <-r.ctx.Done():
			return
		}

		if needFlush {
			// TODO: retry some times?
			if err := r.doBulk(reqs); err != nil {
				log.Errorf("do ES bulk err %v, close sync", err)
				r.cancel()
				return
			}
			reqs = reqs[0:0]
		}

		if needSavePos {
			if err := r.master.Save(pos); err != nil {
				log.Errorf("save sync position %s err %v, close sync", pos, err)
				r.cancel()
				return
			}
		}
	}
}

func (r *River) makeReqColumnData(col *schema.TableColumn, value interface{}) interface{} {
	switch col.Type {
	case schema.TYPE_ENUM:
		switch value := value.(type) {
		case int64:
			// for binlog, ENUM may be int64, but for dump, enum is string
			eNum := value - 1
			if eNum < 0 || eNum >= int64(len(col.EnumValues)) {
				// we insert invalid enum value before, so return empty
				log.Warnf("invalid binlog enum index %d, for enum %v", eNum, col.EnumValues)
				return ""
			}

			return col.EnumValues[eNum]
		}
	case schema.TYPE_SET:
		switch value := value.(type) {
		case int64:
			// for binlog, SET may be int64, but for dump, SET is string
			bitmask := value
			sets := make([]string, 0, len(col.SetValues))
			for i, s := range col.SetValues {
				if bitmask&int64(1<<uint(i)) > 0 {
					sets = append(sets, s)
				}
			}
			return strings.Join(sets, ",")
		}
	case schema.TYPE_BIT:
		switch value := value.(type) {
		case string:
			// for binlog, BIT is int64, but for dump, BIT is string
			// for dump 0x01 is for 1, \0 is for 0
			if value == "\x01" {
				return int64(1)
			}

			return int64(0)
		}
	case schema.TYPE_STRING:
		switch value := value.(type) {
		case []byte:
			return string(value[:])
		}
	case schema.TYPE_JSON:
		var f interface{}
		var err error
		switch v := value.(type) {
		case string:
			err = json.Unmarshal([]byte(v), &f)
		case []byte:
			err = json.Unmarshal(v, &f)
		}
		if err == nil && f != nil {
			return f
		}
	case schema.TYPE_DATETIME, schema.TYPE_TIMESTAMP:
		switch v := value.(type) {
		case string:
			vt, err := time.ParseInLocation(mysql.TimeFormat, string(v), time.Local)
			if err != nil || vt.IsZero() { // failed to parse date or zero date
				return nil
			}
			return vt.Format(time.RFC3339)
		}
	case schema.TYPE_DATE:
		switch v := value.(type) {
		case string:
			vt, err := time.Parse(mysqlDateFormat, string(v))
			if err != nil || vt.IsZero() { // failed to parse date or zero date
				return nil
			}
			return vt.Format(mysqlDateFormat)
		}
	}

	return value
}

func (r *River) getFieldParts(k string, v string) (string, string, string) {
	composedField := strings.Split(v, ",")

	mysql := k
	elastic := composedField[0]
	fieldType := ""

	if 0 == len(elastic) {
		elastic = mysql
	}
	if 2 == len(composedField) {
		fieldType = composedField[1]
	}

	return mysql, elastic, fieldType
}

// If id in toml file is none, get primary keys in one row and format them into a string, and PK must not be nil
// Else get the ID's column in one row and format them into a string
func (r *River) getDocID(rule *Rule, row []interface{}) (string, error) {
	var (
		ids []interface{}
		err error
	)
	if rule.ID == nil {
		ids, err = rule.TableInfo.GetPKValues(row)
		if err != nil {
			return "", err
		}
	} else {
		ids = make([]interface{}, 0, len(rule.ID))
		for _, column := range rule.ID {
			value, err := rule.TableInfo.GetColumnValue(column, row)
			if err != nil {
				return "", err
			}
			ids = append(ids, value)
		}
	}

	var buf bytes.Buffer

	sep := ""
	for i, value := range ids {
		if value == nil {
			return "", errors.Errorf("The %ds id or PK value is nil", i)
		}

		buf.WriteString(fmt.Sprintf("%s%v", sep, value))
		sep = ":"
	}

	return buf.String(), nil
}

func (r *River) getParentID(rule *Rule, row []interface{}, columnName string) (string, error) {
	index := rule.TableInfo.FindColumn(columnName)
	if index < 0 {
		return "", errors.Errorf("parent id not found %s(%s)", rule.TableInfo.Name, columnName)
	}

	return fmt.Sprint(row[index]), nil
}

func (r *River) doBulk(reqs []*elastic.BulkRequest) error {
	if len(reqs) == 0 {
		return nil
	}

	if resp, err := r.es.Bulk(reqs); err != nil {
		log.Errorf("sync docs err %v after binlog %s", err, r.canal.SyncedPosition())
		return errors.Trace(err)
	} else if resp.Code/100 == 2 || resp.Errors {
		for i := 0; i < len(resp.Items); i++ {
			for action, item := range resp.Items[i] {
				if len(item.Error) > 0 {
					log.Errorf("%s index: %s, type: %s, id: %s, status: %d, error: %s",
						action, item.Index, item.Type, item.ID, item.Status, item.Error)
				}
			}
		}
	}

	return nil
}

// get mysql field value and convert it to specific value to es
func (r *River) getFieldValue(col *schema.TableColumn, fieldType string, value interface{}) interface{} {
	var fieldValue interface{}
	switch fieldType {
	case fieldTypeList:
		v := r.makeReqColumnData(col, value)
		if str, ok := v.(string); ok {
			fieldValue = strings.Split(str, ",")
		} else {
			fieldValue = v
		}

	case fieldTypeDate:
		if col.Type == schema.TYPE_NUMBER {
			col.Type = schema.TYPE_DATETIME

			v := reflect.ValueOf(value)
			switch v.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				fieldValue = r.makeReqColumnData(col, time.Unix(v.Int(), 0).Format(mysql.TimeFormat))
			}
		}
	}

	if fieldValue == nil {
		fieldValue = r.makeReqColumnData(col, value)
	}
	return fieldValue
}
