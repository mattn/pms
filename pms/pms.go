package pms

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ambientsound/pms/api"
	"github.com/ambientsound/pms/console"
	"github.com/ambientsound/pms/db"
	"github.com/ambientsound/pms/index"
	"github.com/ambientsound/pms/input"
	"github.com/ambientsound/pms/input/keys"
	"github.com/ambientsound/pms/message"
	pms_mpd "github.com/ambientsound/pms/mpd"
	"github.com/ambientsound/pms/options"
	"github.com/ambientsound/pms/song"
	"github.com/ambientsound/pms/songlist"
	"github.com/ambientsound/pms/style"
	"github.com/ambientsound/pms/term"
	"github.com/ambientsound/pms/widgets"

	"github.com/ambientsound/gompd/mpd"
)

// PMS is a kitchen sink of different objects, glued together as a singleton class.
type PMS struct {
	CLI        *input.CLI
	ui         *widgets.UI
	Options    *options.Options
	Sequencer  *keys.Sequencer
	stylesheet style.Stylesheet
	mutex      sync.Mutex

	// Terminal event interface
	terminal *term.Sampler

	// collection of data
	database *db.Instance

	// MPD connection object
	Connection *Connection

	// Local versions of MPD's queue and song library, in addition to the song library version that was indexed.
	queueVersion   int
	libraryVersion int
	indexVersion   int

	// EventList receives a signal when current songlist has been changed.
	EventList chan int

	// EventLibrary receives a signal when MPD's library has been updated and retrieved.
	EventLibrary chan int

	// EventMessage is used to display text in the statusbar.
	EventMessage chan message.Message

	// EventOption receives a signal when options have been changed.
	EventOption chan string

	// EventPlayer receives a signal when MPD's "player" status changes in an IDLE event.
	EventPlayer chan int

	// EventQueue receives a signal when MPD's "playlist" status changes in an IDLE event.
	EventQueue chan int

	// QuitSignal receives a signal when PMS should quit.
	QuitSignal chan int

	// eventInputCommand sends script lines to the evaluator.
	eventInputCommand chan string
}

func New() (*PMS, error) {
	var err error

	pms := &PMS{}

	pms.database = db.New()

	pms.database.SetQueue(songlist.NewQueue(pms.CurrentMpdClient))
	pms.database.SetLibrary(songlist.NewLibrary())
	pms.database.Panel().Add(pms.database.Queue())
	pms.database.Panel().Add(pms.database.Library())
	pms.database.Panel().Activate(pms.database.Queue())

	pms.EventLibrary = make(chan int, 1024)
	pms.EventList = make(chan int, 1024)
	pms.EventMessage = make(chan message.Message, 1024)
	pms.EventPlayer = make(chan int, 1024)
	pms.EventOption = make(chan string, 1024)
	pms.EventQueue = make(chan int, 1024)
	pms.eventInputCommand = make(chan string, 1024)
	pms.QuitSignal = make(chan int, 1)
	pms.stylesheet = make(style.Stylesheet)

	pms.Options = options.New()
	pms.Options.AddDefaultOptions()

	pms.Sequencer = keys.NewSequencer()

	pms.ui, err = widgets.NewUI(pms.API())
	if err != nil {
		return nil, fmt.Errorf("Error while setting up graphical interface: %s", err)
	}

	pms.CLI = input.NewCLI(pms.API())

	pms.terminal = term.New()

	return pms, nil
}

// StartThreads starts threads that allow PMS to operate asynchronously.
func (pms *PMS) StartThreads() {
	// Terminal input thread
	go pms.terminal.Loop()

	// MPD connection thread
	go pms.Connection.Run()
}

// setupAPI creates an API object
func (pms *PMS) API() api.API {
	return api.BaseAPI(
		pms.Database,
		pms.EventList,
		pms.EventMessage,
		pms.EventOption,
		pms.database.Library,
		pms.CurrentMpdClient,
		pms.Multibar,
		pms.Options,
		pms.database.PlayerStatus,
		pms.database.Queue,
		pms.QuitSignal,
		pms.Sequencer,
		pms.database.CurrentSong,
		pms.CurrentSonglistWidget,
		pms.Stylesheet(),
	)
}
func makeAddress(host, port string) string {
	return fmt.Sprintf("%s:%s", host, port)
}

func (pms *PMS) Message(format string, a ...interface{}) {
	pms.EventMessage <- message.Format(format, a...)
}

func (pms *PMS) Error(format string, a ...interface{}) {
	pms.EventMessage <- message.Errorf(format, a...)
}

// handleConnected (re)synchronizes MPD's state with PMS.
func (pms *PMS) handleConnected() {
	var err error

	console.Log("New connection to MPD.")

	console.Log("Updating current song...")
	err = pms.UpdateCurrentSong()
	if err != nil {
		goto errors
	}

	console.Log("Synchronizing queue...")
	err = pms.SyncQueue()
	if err != nil {
		goto errors
	}

	console.Log("Synchronizing library...")
	err = pms.SyncLibrary()
	if err != nil {
		goto errors
	}

	pms.Message("Ready.")

	return

errors:

	pms.Error("ERROR: %s", err)
	pms.Connection.Close()
}

func (pms *PMS) Database() *db.Instance {
	return pms.database
}

// CurrentMpdClient ensures there is a valid MPD connection, and returns the MPD client object.
func (pms *PMS) CurrentMpdClient() *mpd.Client {
	client, err := pms.Connection.MpdClient()
	if err != nil {
		pms.Error("%s", err)
	}
	return client
}

// CurrentSonglistWidget returns the current songlist.
func (pms *PMS) CurrentSonglistWidget() api.SonglistWidget {
	return pms.ui.Songlist
}

// Stylesheet returns the global stylesheet.
func (pms *PMS) Stylesheet() style.Stylesheet {
	return pms.stylesheet
}

// Multibar returns the multibar widget.
func (pms *PMS) Multibar() api.MultibarWidget {
	return pms.ui.Multibar
}

// UI returns the tcell UI widget.
func (pms *PMS) UI() api.UI {
	return nil
}

// SyncLibrary retrieves the MPD library and stores it as a Songlist in the
// PMS.Library variable. Furthermore, the search index is opened, and if it is
// older than the database version, a reindex task is started.
//
// If the Songlist or Index is cached at the correct version, that part goes untouched.
func (pms *PMS) SyncLibrary() error {
	client, err := pms.Connection.MpdClient()
	if err != nil {
		return err
	}

	stats, err := client.Stats()
	if err != nil {
		return fmt.Errorf("Error while retrieving library stats from MPD: %s", err)
	}

	currentLibrary := pms.database.Library()
	version, _ := strconv.Atoi(stats["db_update"])
	localVersion := currentLibrary.Version()
	console.Log("SyncLibrary(): server reports library version %d", version)
	console.Log("SyncLibrary(): local version is %d", localVersion)

	if version != localVersion {
		console.Log("Switching MPD libraries.")
		console.Log("Closing search index.")
		currentLibrary.CloseIndex()

		console.Log("Retrieving library metadata, %s songs...", stats["songs"])
		library, err := pms.retrieveLibrary()
		if err != nil {
			return fmt.Errorf("Error while retrieving library from MPD: %s", err)
		}

		library.SetVersion(version)
		library.OpenIndex(index.Path(pms.Connection.Host, pms.Connection.Port))

		pms.database.SetLibrary(library)

		console.Log("Library metadata at version %d.", version)

		pms.EventLibrary <- 1
	}

	library := pms.database.Library()
	if !library.IndexSynced() {
		console.Log("Search index is not synchronized with library, rebuilding index...")
		library.ReIndex()
	}

	return nil
}

func (pms *PMS) SyncQueue() error {
	if err := pms.UpdatePlayerStatus(); err != nil {
		return err
	}

	status := pms.database.PlayerStatus()
	if pms.queueVersion == status.Playlist {
		return nil
	}
	queue := pms.database.Queue()

	console.Log("Retrieving changed songs in queue...")
	queueChanges, err := pms.retrieveQueue()
	if err != nil {
		return fmt.Errorf("Error while retrieving queue from MPD: %s", err)
	}
	console.Log("Total of %d changed songs in queue.", queueChanges.Len())
	newQueue, err := queue.Merge(queueChanges)
	if err != nil {
		return fmt.Errorf("Error while merging queue changes: %s", err)
	}
	if err := newQueue.Truncate(status.PlaylistLength); err != nil {
		return fmt.Errorf("Error while truncating queue: %s", err)
	}

	// Replace list while preserving cursor position, either at song ID, or if
	// that failed, place it at the nearest position.
	if err := newQueue.CursorToSong(queue.CursorSong()); err != nil {
		newQueue.SetCursor(queue.Cursor())
	}

	pms.database.SetQueue(newQueue)
	pms.queueVersion = status.Playlist
	console.Log("Queue at version %d.", pms.queueVersion)
	pms.EventQueue <- 1
	return nil
}

func (pms *PMS) retrieveLibrary() (*songlist.Library, error) {
	client, err := pms.Connection.MpdClient()
	if err != nil {
		return nil, err
	}

	timer := time.Now()
	list, err := client.ListAllInfo("/")
	if err != nil {
		return nil, err
	}
	console.Log("ListAllInfo in %s", time.Since(timer).String())

	console.Log("Building library...")

	timer = time.Now()
	s := songlist.NewLibrary()
	s.AddFromAttrlist(list)
	console.Log("Built library in %s", time.Since(timer).String())

	return s, nil
}

func (pms *PMS) retrieveQueue() (*songlist.Queue, error) {
	client, err := pms.Connection.MpdClient()
	if err != nil {
		return nil, err
	}

	timer := time.Now()
	list, err := client.PlChanges(pms.queueVersion, -1, -1)
	if err != nil {
		return nil, err
	}
	console.Log("PlChanges in %s", time.Since(timer).String())

	s := songlist.NewQueue(pms.CurrentMpdClient)
	s.AddFromAttrlist(list)
	return s, nil
}

// UpdateCurrentSong stores a local copy of the currently playing song.
func (pms *PMS) UpdateCurrentSong() error {
	client, err := pms.Connection.MpdClient()
	if err != nil {
		return err
	}

	attrs, err := client.CurrentSong()
	if err != nil {
		return err
	}

	console.Log("MPD current song: %s", attrs["file"])

	s := song.New()
	s.SetTags(attrs)
	pms.database.SetCurrentSong(s)

	pms.EventPlayer <- 0

	return nil
}

// UpdatePlayerStatus populates pms.mpdStatus with data from the MPD server.
func (pms *PMS) UpdatePlayerStatus() error {
	client, err := pms.Connection.MpdClient()
	if err != nil {
		return err
	}

	attrs, err := client.Status()
	if err != nil {
		return err
	}

	status := pms_mpd.PlayerStatus{}
	status.SetTime()

	console.Log("MPD player status: %s", attrs)

	status.Audio = attrs["audio"]
	status.Err = attrs["err"]
	status.State = attrs["state"]

	// The time field is divided into ELAPSED:LENGTH.
	// We only need the length field, since the elapsed field is sent as a
	// floating point value.
	split := strings.Split(attrs["time"], ":")
	if len(split) == 2 {
		status.Time, _ = strconv.Atoi(split[1])
	} else {
		status.Time = -1
	}

	status.Bitrate, _ = strconv.Atoi(attrs["bitrate"])
	status.Playlist, _ = strconv.Atoi(attrs["playlist"])
	status.PlaylistLength, _ = strconv.Atoi(attrs["playlistlength"])
	status.Song, _ = strconv.Atoi(attrs["song"])
	status.SongID, _ = strconv.Atoi(attrs["songid"])
	status.Volume, _ = strconv.Atoi(attrs["volume"])

	status.Elapsed, _ = strconv.ParseFloat(attrs["elapsed"], 64)
	status.ElapsedPercentage, _ = strconv.ParseFloat(attrs["elapsedpercentage"], 64)
	status.MixRampDB, _ = strconv.ParseFloat(attrs["mixrampdb"], 64)

	status.Consume, _ = strconv.ParseBool(attrs["consume"])
	status.Random, _ = strconv.ParseBool(attrs["random"])
	status.Repeat, _ = strconv.ParseBool(attrs["repeat"])
	status.Single, _ = strconv.ParseBool(attrs["single"])

	pms.EventPlayer <- 0

	// Make sure any error messages are relayed to the user
	if len(attrs["error"]) > 0 {
		pms.Error(attrs["error"])
	}

	pms.database.SetPlayerStatus(status)

	return nil
}

func (pms *PMS) Execute(cmd string) {
	console.Log("Execute command: '%s'", cmd)
	err := pms.CLI.Execute(cmd)
	if err != nil {
		pms.Error("%s", err)
	}
}
