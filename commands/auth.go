package commands

import (
	"github.com/ambientsound/pms/api"
)

// Auth runs OAuth2 authentication flow against Spotify.
type Auth struct {
	newcommand
	api api.API
}

func NewAuth(api api.API) Command {
	return &Auth{
		api: api,
	}
}

// Parse implements Command.
func (cmd *Auth) Parse() error {
	return cmd.ParseEnd()
}

func (cmd *Auth) Exec() error {
	cmd.api.Authenticate()
	return nil
}