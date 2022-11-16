package main

import (
	"fmt"
	"strings"
)

type URI struct {
	Type    string
	SubType string
	Rest    string
}

func (u *URI) String() string {
	if u.SubType != "" {
		return fmt.Sprintf("%s+%s://%s", u.Type, u.SubType, u.Rest)
	}
	return fmt.Sprintf("%s://%s", u.Type, u.Rest)
}

func ParseURI(uri string) (*URI, error) {
	parts := strings.Split(uri, "://")
	if len(parts) == 2 {
		types := strings.Split(parts[0], "+")
		if len(types) == 2 {
			return &URI{
				Type:    strings.ToLower(types[0]),
				SubType: strings.ToLower(types[1]),
				Rest:    parts[1],
			}, nil
		}
		return &URI{Type: parts[0], Rest: parts[1]}, nil
	}
	return nil, fmt.Errorf("invalid uri: %s", uri)
}
