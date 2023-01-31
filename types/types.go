package types

import (
	"errors"
	"os"
	"strings"
)

// Bool represents a native `bool` type
type Bool bool

// UnmarshalText satisfies the encoding.TextUnmarshaler interface
func (b *Bool) UnmarshalText(text []byte) error {
	switch string(text) {
	case "true":
		*b = true
	case "false":
		*b = false
	default:
		return errors.New("text is not a boolean")
	}

	return nil
}

// SliceString represents a string slice
type SliceString []string

// UnmarshalText satisfies the encoding.TextUnmarshaler interface
func (s *SliceString) UnmarshalText(text []byte) error {
	*s = strings.Split(string(text), ",")
	return nil
}

// MapString represents a string map of strings
type MapString map[string]string

// UnmarshalText satisfies the encoding.TextUnmarshaler interface
func (m *MapString) UnmarshalText(b []byte) (err error) {
	var (
		text = string(b)
	)

	text, err = m.filetext(text)
	if err != nil {
		return err
	}

	r := make(MapString)

	for _, line := range strings.Split(strings.Replace(text, ",", "\n", -1), "\n") {
		if split := strings.Split(line, "="); len(split) > 1 {
			if r[split[0]], err = m.filetext(string(split[1])); err != nil {
				return err
			}
		}
	}

	*m = r

	return nil
}

// filetext treats `text` as a filepath and returns the content or returns a literal string
func (m *MapString) filetext(text string) (s string, err error) {
	_, err = os.Stat(text)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return s, err
	}

	if err == nil {
		return m.file(text)
	}

	return text, nil
}

// file reads the file at the given path and returns the content as a string
func (m *MapString) file(path string) (s string, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}

	return string(b), nil
}
