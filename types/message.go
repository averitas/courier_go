package types

import "encoding/json"

type Message struct {
	Code    int
	Message string
}

func DeserilizeMessage(b []byte) (*Message, error) {
	msg := &Message{}
	err := json.Unmarshal(b, &msg)
	return msg, err
}

func (m *Message) Serilize() ([]byte, error) {
	return json.Marshal(m)
}
