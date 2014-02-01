// Package visits provides a barebones mechanic to greet players.
// Only the joining player will see this greeting and the number
// of times he has joined.
package cod

import (
	"database/sql"
	"fmt"
	integrated "github.com/adabei/goldenbot-integrated/cod"
	"github.com/adabei/goldenbot/events"
	"github.com/adabei/goldenbot/events/cod"
	"github.com/adabei/goldenbot/rcon"
	"log"
	"strconv"
)

type VisitsConfig struct {
	Prefix       string
	FirstMessage string
	Message      string
}

const schema = `
create table visits (
  players_id text primary key,
  total integer default 1
);`

type Visits struct {
	Config   VisitsConfig
	requests chan rcon.RCONQuery
	events   chan interface{}
	db       *sql.DB
}

func NewVisits(config VisitsConfig, requests chan rcon.RCONQuery, ea events.Aggregator, db *sql.DB) *Visits {
	v := new(Visits)
	v.Config = config
	v.requests = requests
	v.events = ea.Subscribe(v)
	v.db = db
	return v
}

func (v *Visits) Setup() error {
	_, err := v.db.Exec(schema)
	return err
}

func (v *Visits) Start() {
	for {
		in := <-v.events
		if ev, ok := in.(cod.Join); ok {
			if exists(v.db, ev.GUID) {
				// update
				log.Println("visits: updating total for player", ev.GUID)
				_, err := v.db.Exec("update visits set total = total + 1 where players_id = ?;", ev.GUID)

				if err != nil {
					log.Fatal("visits: fatal error in update:", err)
				}
			} else {
				// insert new
				log.Println("visits: inserting player", ev.GUID, "into database")
				_, err := v.db.Exec("insert into visits(players_id) values(?);", ev.GUID)

				if err != nil {
					log.Fatal("visits: fatal error in insert:", err)
				}
			}

			var total int
			err := v.db.QueryRow("select total from visits where players_id = ?", ev.GUID).Scan(&total)
			if err != nil {
				total = 1
			}

			var msg string
			if total != 1 {
				msg = fmt.Sprintf(v.Config.Message, ev.Name, total)
			} else {
				msg = fmt.Sprintf(v.Config.FirstMessage, ev.Name)
			}

			if num, ok := integrated.Num(ev.GUID); ok {
				log.Println("visits: welcoming player with guid", ev.GUID, "and num", num)
				v.requests <- rcon.RCONQuery{Command: "tell " + strconv.Itoa(num) + " " +
					v.Config.Prefix + msg, Response: nil}
			} else {
				log.Println("visits: could not resolve num for player", ev.GUID)
			}
		}
	}
}

func exists(db *sql.DB, id string) bool {
	var guid string
	err := db.QueryRow("select players_id from visits where players_id = ?", id).Scan(&guid)
	if err != nil {
		return false
	}

	return true
}
