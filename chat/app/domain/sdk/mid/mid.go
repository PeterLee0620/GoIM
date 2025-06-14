// Package mid provides app level middleware support.
package mid

import "github.com/DavidLee0620/GoIM/chat/foundation/web"

// isError tests if the Encoder has an error inside of it.
func isError(e web.Encoder) error {
	err, isError := e.(error)
	if isError {
		return err
	}
	return nil
}
