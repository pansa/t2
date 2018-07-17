package main

import (
	"errors"
	"path"
	"testing"
)

func TestNewConfig(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{
			name: "not found",
			err:  errors.New("open mocks/config/not found: no such file or directory"),
		},
		{
			name: "error.json",
			err:  errors.New("json: cannot unmarshal number into Go struct field Config.host of type string"),
		},
		{
			name: "empty.json",
			err:  nil,
		},
	}

	for _, tc := range cases {
		_, err := NewConfig(path.Join("./mocks/config", tc.name))

		if tc.err != nil {
			if err == nil || err.Error() != tc.err.Error() {
				t.Errorf("Error must be %v but got %v\n", tc.err, err)
			}

			continue
		}

		if err != nil {
			t.Errorf("Error must be nil but got %v\n", err)
		}
	}
}
