package options

import (
	"fmt"
	"github.com/spf13/viper"
)

// Option names.
const (
	Center  = "center"
	Columns = "columns"
	Limit   = "limit"
	Sort    = "sort"
	Topbar  = "topbar"

	SpotifyClientID     = "spotifyclientid"
	SpotifyClientSecret = "spotifyclientsecret"

	LogFile      = "logfile"
	LogOverwrite = "logoverwrite"
)

// Option types.
const (
	boolType   = false
	intType    = 0
	stringType = ""
)

// Initialize option types.
// Default values must be defined in the Defaults string.
func init() {
	viper.Set(Center, boolType)
	viper.Set(Columns, stringType)
	viper.Set(Limit, intType)
	viper.Set(LogFile, stringType)
	viper.Set(LogOverwrite, boolType)
	viper.Set(Sort, stringType)
	viper.Set(SpotifyClientID, stringType)
	viper.Set(SpotifyClientSecret, stringType)
	viper.Set(Topbar, stringType)
}

// Return a human-readable representation of an option.
// This string can be used in a config file.
func Print(key string, opt interface{}) string {
	switch v := opt.(type) {
	case string:
		return fmt.Sprintf("%s=\"%s\"", key, v)
	case int:
		return fmt.Sprintf("%s=%d", key, v)
	case bool:
		if !v {
			return fmt.Sprintf("no%s", key)
		}
		return fmt.Sprintf("%s", key)
	default:
		return fmt.Sprintf("%s=%v", key, v)
	}
}

// Default configuration file.
const Defaults string = `
# Global options
set nocenter
set columns=artist,track,title,album,year,time,popularity
set sort=track,disc,album,year,artist
set topbar="|$shortname $version||;${tag|artist} - ${tag|title}||${tag|album}, ${tag|year};$volume $mode $elapsed ${state} $time;|[${list|index}/${list|total}] ${list|title}||;;"
set limit=50

# Logging
set nologoverwrite
set logfile=

# Song tag styles
style album teal
style artist yellow
style date green
style time darkmagenta
style title white bold
style disc darkgreen
style track green
style year green
style originalyear darkgreen

# Tracklist styles
style allTagsMissing red
style currentSong black yellow
style cursor black white
style header teal bold
style mostTagsMissing red
style selection white blue

# Topbar styles
style elapsedTime green
style elapsedPercentage green
style listIndex darkblue
style listTitle blue bold
style listTotal darkblue
style mute red
style shortName bold
style state default
style switches teal
style tagMissing red
style topbar darkgray
style version gray
style volume green

# Other styles
style commandText default
style errorText white red bold
style logLevel white
style logMessage gray
style readout default
style searchText white bold
style sequenceText teal
style statusbar default
style timestamp teal
style visualText teal

# Keyboard bindings: cursor and viewport movement
bind <Up> cursor up
bind k cursor up
bind <Down> cursor down
bind j cursor down
bind <PgUp> viewport pgup
bind <PgDn> viewport pgdn
bind <C-b> viewport pgup
bind <C-f> viewport pgdn
bind <C-u> viewport halfpgup
bind <C-d> viewport halfpgdn
bind <C-y> viewport up
bind <C-e> viewport down
bind <Home> cursor home
bind gg cursor home
bind <End> cursor end
bind G cursor end
bind gc cursor current
bind R cursor random
bind b cursor prevOf album
bind e cursor nextOf album
bind H cursor high
bind M cursor middle
bind L cursor low
bind zb viewport high
bind z- viewport high
bind zz viewport middle
bind z. viewport middle
bind zt viewport low
bind z<Enter> viewport low

# Keyboard bindings: input mode
bind : inputmode input
bind / inputmode search
bind <F3> inputmode search
bind v select visual
bind V select visual

# Keyboard bindings: player and mixer
bind <Enter> play selection
bind <Space> pause
bind s stop
bind h previous
bind l next
bind + volume +2
bind - volume -2
bind <left> seek -5
bind <right> seek +5
bind <Alt-M> volume mute
bind S single

# Keyboard bindings: other
bind <C-c> quit
bind <C-l> redraw
bind <C-s> sort
bind i print file
bind gt list next
bind gT list previous
bind t list next
bind T list previous
bind <C-w>d list duplicate
bind <C-g> list remove
bind <C-j> isolate artist
bind <C-t> isolate albumartist album
bind & select nearby albumartist album
bind m select toggle
bind a add
bind <Delete> cut
bind x cut
bind y yank
bind p paste after
bind P paste before
`
