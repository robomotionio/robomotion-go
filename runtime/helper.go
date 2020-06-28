package runtime

import (
	"fmt"
)

var (
	runtimeHelper RuntimeHelper
)

func Close() error {
	return runtimeHelper.Close()
}

func EmitDebug(guid, name string, message interface{}) error {
	if runtimeHelper == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.Debug(guid, name, message)
}
