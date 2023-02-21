package main

import (
	"encoding/json"
	"os"
)

type State struct {
	filename      string
	MaxId         int `json:"maxid"`
	writeRequired bool
}

func NewState(fname string) *State {
	s := State{
		filename: fname,
		MaxId:    0,
	}
	s.Load()
	return &s
}

func (s *State) Load() *State {
	data, _ := os.ReadFile(s.filename)
	json.Unmarshal(data, &s)
	return s
}

func (s *State) Save() {
	if !s.writeRequired {
		return
	}
	s.writeRequired = false
	jsonString, _ := json.Marshal(s)
	os.WriteFile(s.filename, jsonString, 0644)
}

func (s *State) SetMaxId(maxid int) *State {
	if s.MaxId != maxid {
		s.writeRequired = true
	}
	s.MaxId = maxid
	return s
}
