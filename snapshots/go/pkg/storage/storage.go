/* Copyright 2022 Cognite AS */

package storage

import (
	_ "go.beyondstorage.io/services/gcs/v3"
	"go.beyondstorage.io/v5/services"
	"go.beyondstorage.io/v5/types"
)

var IteratorDone = types.IterateDone

func NewStorage(storageUrl string) (types.Storager, error) {
	return services.NewStoragerFromString(storageUrl)
}
