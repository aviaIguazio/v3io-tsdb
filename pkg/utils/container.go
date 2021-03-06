/*
Copyright 2018 Iguazio Systems Ltd.

Licensed under the Apache License, Version 2.0 (the "License") with
an addition restriction as set forth herein. You may not use this
file except in compliance with the License. You may obtain a copy of
the License at http://www.apache.org/licenses/LICENSE-2.0.

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
implied. See the License for the specific language governing
permissions and limitations under the License.

In addition, you may not use the software for any purposes that are
illegal under applicable law, and the grant of the foregoing license
under the Apache 2.0 license is conditioned upon your compliance with
such restriction.
*/

package utils

import (
	"encoding/binary"
	"github.com/nuclio/logger"
	"github.com/nuclio/zap"
	"github.com/pkg/errors"
	"github.com/v3io/v3io-go-http"
)

func NewLogger(verbose string) (logger.Logger, error) {
	var logLevel nucliozap.Level
	switch verbose {
	case "debug":
		logLevel = nucliozap.DebugLevel
	case "info":
		logLevel = nucliozap.InfoLevel
	case "warn":
		logLevel = nucliozap.WarnLevel
	case "error":
		logLevel = nucliozap.ErrorLevel
	default:
		logLevel = nucliozap.InfoLevel
	}

	log, err := nucliozap.NewNuclioZapCmd("v3io-prom", logLevel)
	if err != nil {
		return nil, err
	}
	return log, nil
}

func CreateContainer(logger logger.Logger, addr, cont string, workers int) (*v3io.Container, error) {
	// create context
	context, err := v3io.NewContext(logger, addr, workers)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create client")
	}

	// create session
	session, err := context.NewSession("", "", "v3test")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create session")
	}

	// create the container
	container, err := session.NewContainer(cont)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create container")
	}

	return container, nil
}

// convert v3io blob to Int array
func AsInt64Array(val []byte) []uint64 {
	var array []uint64
	bytes := val
	for i := 16; i+8 <= len(bytes); i += 8 {
		val := binary.LittleEndian.Uint64(bytes[i : i+8])
		array = append(array, val)
	}
	return array
}


func DeleteTable(container *v3io.Container, path string) error {

	input := v3io.GetItemsInput{ Path: path, AttributeNames: []string{"__name"}}
	iter, err := container.Sync.GetItemsCursor(&input)
	if err != nil {
		return err
	}

	responseChan := make(chan *v3io.Response, 1000)
	reqMap := map[uint64]bool{}

	for iter.Next() {
		name := iter.GetField("__name").(string)
		req, err := container.DeleteObject(&v3io.DeleteObjectInput{Path: path +"/" + name}, nil, responseChan)
		if err != nil {
			return errors.Wrap(err, "failed to delete object " + name)
		}
		reqMap[req.ID] = true
	}

	for len(reqMap) > 0 {
		select {
		case resp := <-responseChan:
			if resp.Error != nil {
				return errors.Wrap(err, "failed Delete response")
			}

			delete(reqMap, resp.ID)
		}
	}

	return nil
}