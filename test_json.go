package main

import (
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func main() {
	id := bson.NewObjectID()
	m := bson.M{
		"_id":  id,
		"name": "test",
	}

	data, err := json.Marshal(m)
	if err != nil {
		fmt.Printf("Json Marshal error: %v\n", err)
		return
	}

	fmt.Printf("Json Result: %s\n", string(data))
}
