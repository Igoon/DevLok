//go:build !windows && !darwin

package tlsca

import "errors"

func trustCA(string) error { return errors.New("trust CA belum tersedia di platform ini") }

func untrustCA(string) error { return errors.New("untrust CA belum tersedia di platform ini") }
