package main

import "encoding/json"

type Storage struct {
	PersistentStorage map[string]interface{}
	TemporaryStorage  map[string]interface{}
	StorageHistory    []map[string]interface{}
	StateHistory      []string
}

func (s *Storage) PushHistory(state string) error {
	// Clone by json
	var clone map[string]interface{}

	jsonStr, err := json.Marshal(&s.PersistentStorage)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(jsonStr, &clone); err != nil {
		return err
	}

	s.StorageHistory = append(s.StorageHistory, clone)

	return nil
}

func (s *Storage) Clear() {
	s.PersistentStorage = make(map[string]interface{})
	s.TemporaryStorage = make(map[string]interface{})
}
