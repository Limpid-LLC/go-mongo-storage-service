# saiStorageMongo

## Ports
`8801`: **HTTP**  

## How to run
`make build`: rebuild and start service  
`make up`: start service  
`make down`: stop service  
`make logs`: display service logs  

## API
### Save data
#### Request: ${SERVER_URL}/save
```json
{
  "collection": "CollectionName",
  "data": {
    "field1": "value1",
    "field2": "value2"
  }
} 
```

#### Response: 
```json
{
  "Status": "Ok",
  "Result": ${created_internal_id}
}
```

### Get data
#### Request: ${SERVER_URL}/get
```json
{
  "collection": "CollectionName",
  "select": {
    "field1": "value1",
    "field2": "value2"
    // any mongo request: $and, $or, $elemntMatch ...
  },
  "options": {
    "$sort": {"${field_name}": 1 or 0},
    "$skip": ${amount_to_skip},
    "$limit": ${amount_to limit},
    "$count": 1 <- to get total count in the result, does not work with limit
  }
} 
```

#### Response: 
```json
{
  "Status": "Ok",
  "Result": []Documents
}
```
