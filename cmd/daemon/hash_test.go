package main

import (
	"bytes"
	"io"
	"testing"
	"testing/iotest"
)

func TestGetMd5Sum(t *testing.T) {
	cases := []struct {
		hasError bool
		strData  string
		res      string
	}{
		{
			hasError: true,
			strData:  "example",
		},
		{
			hasError: false,
			strData:  "example",
			res:      "1a79a4d60de6718e8e5b326e338ae533",
		},
	}

	for _, tc := range cases {
		var r io.Reader
		if tc.hasError {
			r = iotest.TimeoutReader(bytes.NewBuffer([]byte(tc.strData)))
		} else {
			r = bytes.NewBuffer([]byte(tc.strData))
		}

		str, err := getMd5Sum(r)

		if tc.hasError {
			if err == nil {
				t.Error("Error must be not nil")
			}

			if str != "" {
				t.Error("Result must be empty string")
			}

			continue
		}

		if err != nil {
			t.Errorf("Error must be nil but got %v", err)
		}

		if str != tc.res {
			t.Errorf("Result must %v but got %v", tc.res, str)
		}
	}
}

func TestGetSha1Sum(t *testing.T) {
	cases := []struct {
		hasError bool
		strData  string
		res      string
	}{
		{
			hasError: true,
			strData:  "example",
		},
		{
			hasError: false,
			strData:  "example",
			res:      "c3499c2729730a7f807efb8676a92dcb6f8a3f8f",
		},
	}

	for _, tc := range cases {
		var r io.Reader
		if tc.hasError {
			r = iotest.TimeoutReader(bytes.NewBuffer([]byte(tc.strData)))
		} else {
			r = bytes.NewBuffer([]byte(tc.strData))
		}

		str, err := getSHA1Sum(r)

		if tc.hasError {
			if err == nil {
				t.Error("Error must be not nil")
			}

			if str != "" {
				t.Error("Result must be empty string")
			}

			continue
		}

		if err != nil {
			t.Errorf("Error must be nil but got %v", err)
		}

		if str != tc.res {
			t.Errorf("Result must %v but got %v", tc.res, str)
		}
	}
}

func TestGetSha256Sum(t *testing.T) {
	cases := []struct {
		hasError bool
		strData  string
		res      string
	}{
		{
			hasError: true,
			strData:  "example",
		},
		{
			hasError: false,
			strData:  "example",
			res:      "50d858e0985ecc7f60418aaf0cc5ab587f42c2570a884095a9e8ccacd0f6545c",
		},
	}

	for _, tc := range cases {
		var r io.Reader
		if tc.hasError {
			r = iotest.TimeoutReader(bytes.NewBuffer([]byte(tc.strData)))
		} else {
			r = bytes.NewBuffer([]byte(tc.strData))
		}

		str, err := getSHA256Sum(r)

		if tc.hasError {
			if err == nil {
				t.Error("Error must be not nil")
			}

			if str != "" {
				t.Error("Result must be empty string")
			}

			continue
		}

		if err != nil {
			t.Errorf("Error must be nil but got %v", err)
		}

		if str != tc.res {
			t.Errorf("Result must %v but got %v", tc.res, str)
		}
	}
}
