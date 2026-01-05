package security

import (
	"errors"
	"os"
)

func RequireRoot(feature string) error {
	if os.Geteuid() != 0 {
		return errors.New(feature + " requires elevated privileges.")
	}

	return nil
}
