package upload

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type Uploads struct {
	UserId string        `json:"userId,omitempty" gorm:"column:userId;primary_key" bson:"_id,omitempty" dynamodbav:"userId,omitempty" firestore:"userId,omitempty" validate:"max=40"`
	Data   []FileUploads `json:"data,omitempty" gorm:"column:data" bson:"data,omitempty" dynamodbav:"data,omitempty" firestore:"data,omitempty"`
}

type FileUploads struct {
	Source   string `json:"source,omitempty" gorm:"column:source" bson:"_source,omitempty" dynamodbav:"source,omitempty" firestore:"source,omitempty"`
	Type 	 string `json:"type,omitempty" gorm:"column:type" bson:"type,omitempty" dynamodbav:"type,omitempty" firestore:"type,omitempty"`
	Url      string `json:"url,omitempty" gorm:"column:url" bson:"url,omitempty" dynamodbav:"url,omitempty" firestore:"url,omitempty"`
}

func (c FileUploads) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *FileUploads) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &c)
}
