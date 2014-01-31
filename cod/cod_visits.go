// Package visits provides a barebones mechanic to greet players.
package cod

import (
	"database/sql"
	"fmt"
	integrated "github.com/adabei/goldenbot-integrated/cod"
	"github.com/adabei/goldenbot/events"
	"github.com/adabei/goldenbot/events/cod"
	"github.com/adabei/goldenbot/helpers"
	"github.com/adabei/goldenbot/rcon"
	"log"
)

var config map[string]interface{} = map[string]interface{}{"message": "Welcome to the server %v. You have visited %d times."}

const schema = `
create table visits (
  players_id text primary key,
  total integer default 1
);`

type Visits struct {
	cfg      map[string]interface{}
	requests chan rcon.RCONQuery
	events   chan interface{}
	db       *sql.DB
}

func NewVisits(cfg map[string]interface{}, requests chan rcon.RCONQuery, ea events.Aggregator, db *sql.DB) *Visits {
	v := new(Visits)
	v.cfg = make(map[string]interface{})
	helpers.Merge(config, v.cfg)
	helpers.Merge(cfg, v.cfg)
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
				_, err := v.db.Exec("update visits set total = total + 1 where players_id = ?;", ev.GUID)

				if err != nil {
					log.Fatal(err)
				}
			} else {
				// insert new
				_, err := v.db.Exec("insert into visits(players_id) values(?);", ev.GUID)

				if err != nil {
					log.Fatal(err)
				}
			}

			var total int
			err := v.db.QueryRow("select total from visits where players_id = ?", ev.GUID).Scan(&total)
			if err != nil {
				total = 1
			}

			msg, ok := v.cfg["message"].(string)
			if ok {
				num := integrated.Num(ev.GUID)
				if num != -1 {
					v.requests <- rcon.RCONQuery{Command: "tell " + string(num) + " \"" +
						fmt.Sprintf(msg, ev.Name, total) + "\"", Response: nil}
				} else {
					log.Println("Could not resolve Num for GUID ", ev.GUID, ".")
				}
			} else {
				log.Println("Could not cast welcome message. No such messages will be sent.")
			}
		}
	}
}

func exists(db *sql.DB, id string) bool {
  var guid string
  err := db.QueryRow("select players_id from visits where players_id = ?", id).Scan(&guid)
  if err != nil {
    log.Println(err)
    return false
  }

  return true
}
