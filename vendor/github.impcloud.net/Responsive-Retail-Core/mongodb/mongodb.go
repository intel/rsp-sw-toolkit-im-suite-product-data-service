package mongodb

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/globalsign/mgo"
)

// DB is a collection of support for different DB technologies. Currently
// only MongoDB has been implemented. We want to be able to access the raw
// database support for the given DB so an interface does not work. Each
// database is too different.
type DB struct {

	// MongoDB Support.
	database *mgo.Database
	session  *mgo.Session
}

// NewSession returns a new DB value for use with MongoDB based on a registered
// master session.
func NewSession(url string, timeout time.Duration) (*DB, error) {
	session, err := newSession(url, timeout)
	if err != nil {
		return nil, err
	}

	db := DB{
		database: session.DB(""),
		session:  session,
	}

	return &db, nil
}

// newMGO creates a new mongo connection. If no url is provided,
// it will default to localhost:27017.
func newSession(url string, timeout time.Duration) (*mgo.Session, error) {

	// Set the default timeout for the session.
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	// Create a session which maintains a pool of socket connections
	// to our MongoDB.
	ses, err := mgo.DialWithTimeout(url, timeout)
	if err != nil {
		return nil, err
	}

	// Reads may not be entirely up-to-date, but they will always see the
	// history of changes moving forward, the data read will be consistent
	// across sequential queries in the same session, and modifications made
	// within the session will be observed in following queries (read-your-writes).
	// http://godoc.org/labix.org/v2/mgo#Session.SetMode
	ses.SetMode(mgo.Monotonic, true)

	return ses, nil
}

// CopySession returns a new DB value for use with MongoDB based the master session.
func (db *DB) CopySession() *DB {

	copySession := db.session.Copy()

	// As per the mgo documentation, https://godoc.org/gopkg.in/mgo.v2#Session.DB
	// if no database name is specified, then use the default one, or the one that
	// the connection was dialed with.

	newDB := DB{
		database: copySession.DB(""),
		session:  copySession,
	}

	return &newDB
}

// Close closes a DB value being used with MongoDB.
func (db *DB) Close() {
	db.session.Close()
}

// Execute is used to execute MongoDB commands.
func (db *DB) Execute(collName string, f func(*mgo.Collection) error) error {
	if db == nil || db.session == nil {
		return errors.New("db == nil || db.session == nil")
	}

	return f(db.database.C(collName))
}

// ExecuteWithChangeInfo is used to execute MongoDB commands.
func (db *DB) ExecuteWithChangeInfo(collName string, f func(*mgo.Collection) (*mgo.ChangeInfo, error)) (*mgo.ChangeInfo, error) {
	if db == nil || db.session == nil {
		return nil, errors.New("db == nil || db.session == nil")
	}

	return f(db.database.C(collName))
}

// ExecuteBulk is used to execute Bulk commands.
func (db *DB) ExecuteBulk(collName string, f func(*mgo.Collection) *mgo.Bulk) *mgo.Bulk {
	return f(db.database.C(collName))
}

// ExecuteCount returns the total number of documents in the collection.
func (db *DB) ExecuteCount(collName string, f func(*mgo.Collection) (int, error)) (int, error) {
	if db == nil || db.session == nil {
		return -1, errors.New("db == nil || db.session == nil")
	}

	return f(db.database.C(collName))
}

// QueryToString provides a string version of the value
func QueryToString(value interface{}) string {
	json, err := json.Marshal(value)
	if err != nil {
		return ""
	}

	return string(json)
}
