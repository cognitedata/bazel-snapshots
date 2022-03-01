/* Copyright 2022 Cognite AS */

package storage

import (
	"io/ioutil"

	"github.com/graymeta/stow"
)

type (
	Item stow.Item
)

// Dial dials to the stow storage.
func Dial(kind string, config stow.Config) (stow.Location, error) {
	return stow.Dial(kind, config)
}

// ToString is a utility function for converting `stow.Item`'s data to
// human readable string.
func ToString(item stow.Item) (string, error) {
	stream, err := item.Open()
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadAll(stream)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Walk walks through each object inside the bucket.
func Walk(bucket stow.Container, prefix string, pageSize int, fn stow.WalkFunc) error {
	return stow.Walk(bucket, prefix, pageSize, fn)
}
