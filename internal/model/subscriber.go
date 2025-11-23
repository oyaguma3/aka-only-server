package model

import "time"

type Subscriber struct {
	IMSI      string    `json:"imsi" db:"imsi"`
	Ki        string    `json:"ki" db:"ki"`
	Opc       string    `json:"opc" db:"opc"`
	SQN       string    `json:"sqn" db:"sqn"`
	AMF       string    `json:"amf" db:"amf"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
