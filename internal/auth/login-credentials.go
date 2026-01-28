package auth

import (
	"context"
	"fmt"
)

func (a *Auth) passwordLogin(ctx context.Context) (string, string, int64, error) {
	return "", "", 0, fmt.Errorf("Not Implemented")
}

// refreshPasswordToken re-authenticates using stored credentials
func (a *Auth) refreshPasswordToken(ctx context.Context, encodedCredentials string) (string, string, int64, error) {
	return "", "", 0, fmt.Errorf("Not Implemented")
}
