/* Copyright 2022 Cognite AS */

package cache

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestDialTargetWithOptions(t *testing.T) {
	var conn *grpc.ClientConn
	var err error

	// illegal scheme
	conn, err = DialTargetWithOptions("wrongscheme://some-uri", false, nil)
	require.ErrorIs(t, err, ErrScheme)
	require.Nil(t, conn)

	// legal scheme
	conn, err = DialTargetWithOptions("bytestream://some-uri", false, nil)
	require.Nil(t, err)
	require.NotNil(t, conn)
}
