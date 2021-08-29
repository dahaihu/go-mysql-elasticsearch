
# nested field create, update, delete
curl -X PUT -H 'Content-Type: application/json' -d '
{
    "mappings": {
         "properties" : {
             "user_role" : {
                 "type" : "nested",
                 "properties" : {
                     "user_id" : { "type" : "long" },
                     "role_id" : { "type" : "long" }
                 }
             }
         }
    }
}' 'http://localhost:9200/resource'

curl -X POST -H 'Content-Type: application/json' -d '{
    "name":"zhangsan",
    "user_role":[{
        "user_id":1,
        "role_id":2
    }]
}' 'http://localhost:9200/resource/_doc/1'
curl -XPOST -H 'Content-type: application/json' -d '{"script":{"lang": "painless","params":{"item":{"role_id":1,"user_id":10}},"source":"ctx._source.user_role.add(params.item)"}}' localhost:9200/resource/_doc/1/_update
curl -XPOST -H 'Content-type: application/json' -d '{"script":{"params":{"user_id":10},"source":"ctx._source.user_role.removeIf(item -\u003e item.user_id == params.user_id)"}}' localhost:9200/resource/_doc/1/_update
curl -XPOST -H 'Content-type: application/json' -d '{"script":{"params":{"role_id":100,"user_id":100},"source":"ctx._source.user_role.removeIf(item -\u003e item.user_id == params.user_id); ctx._source.user_role.add(params)"}}' localhost:9200/resource/_doc/1/_update
