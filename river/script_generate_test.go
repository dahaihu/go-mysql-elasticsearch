package river

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
)

func TestMakeNestedFieldDeleteRequest(t *testing.T) {
	data := makeNestedFieldDelRequest(
		"user_role", "user_id",
		map[string]interface{}{"user_id": 10},
	)
	fmt.Println("data is ", data)
	dataS, _ := json.Marshal(data)
	fmt.Println(
		string(dataS),
	)
}

func TestMakeNestedFieldInsertRequest(t *testing.T) {
	data := makeNestedFieldInsertRequest(
		"user_role",
		map[string]interface{}{"user_id": 10, "role_id": 1},
	)
	fmt.Println("data is ", data)
	dataS, _ := json.Marshal(data)
	fmt.Println(
		string(dataS),
	)
}

func TestMakeNestedFieldUpdateRequest(t *testing.T) {
	data := makeNestedFieldUpdateRequest(
		"user_role", "user_id",
		map[string]interface{}{"user_id": 10, "role_id": 10},
	)
	fmt.Println("data is ", data)
	dataS, _ := json.Marshal(data)
	fmt.Println(
		string(dataS),
	)
}

func TestMatch(t *testing.T) {
	fmt.Println(regexp.QuoteMeta(`Escaping symbols like: .+*?()|[]{}^$`))
}

