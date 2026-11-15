package utils

import (
	"errors"
	"strings"
)

var (
	ErrEmptyName = errors.New("name must be a non-empty string")
	ErrEmptyTag  = errors.New("tags must contain only non-empty strings")
)

func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrEmptyName
	}
	return name, nil
}

func NormalizeTags(tags []string) ([]string, error) {
	if tags == nil {
		return []string{}, nil
	}

	result := make([]string, len(tags))
	copy(result, tags)

	for _, tag := range tags {
		if tag == "" {
			return nil, ErrEmptyTag
		}
	}

	return result, nil
}
