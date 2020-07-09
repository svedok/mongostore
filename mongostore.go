// Copyright 2012 The KidStuff Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mongostore

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Session object store in MongoDB
type Session struct {
	Id       primitive.ObjectID `bson:"_id,omitempty"`
	Data     string
	Modified time.Time
}

// MongoStore stores sessions in MongoDB
type MongoStore struct {
	Codecs  []securecookie.Codec
	Options *sessions.Options
	Token   TokenGetSeter
	coll    *mongo.Collection
}

// NewMongoStore returns a new MongoStore.
// Set ensureTTL to true let the database auto-remove expired object by maxAge.
func NewMongoStore(c *mongo.Collection, maxAge int, ensureTTL bool, keyPairs ...[]byte) (*MongoStore, error) {
	store := &MongoStore{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: maxAge,
		},
		Token: &CookieToken{},
		coll:  c,
	}

	store.MaxAge(maxAge)

	if ensureTTL {
		o := options.Index()
		o.SetBackground(true)
		o.SetSparse(true)
		o.SetExpireAfterSeconds(int32(maxAge))

		_, err := c.Indexes().CreateOne(
			context.Background(),
			mongo.IndexModel{
				Keys: bson.M{
					"modified": 1,
				},
				Options: o,
			},
		)

		if err != nil {
			return nil, err
		}
	}

	return store, nil
}

// Get registers and returns a session for the given name and session store.
// It returns a new session if there are no sessions registered for the name.
func (m *MongoStore) Get(r *http.Request, name string) (
	*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(m, name)
}

// New returns a session for the given name without adding it to the registry.
func (m *MongoStore) New(r *http.Request, name string) (
	*sessions.Session, error) {
	session := sessions.NewSession(m, name)
	session.Options = &sessions.Options{
		Path:     m.Options.Path,
		MaxAge:   m.Options.MaxAge,
		Domain:   m.Options.Domain,
		Secure:   m.Options.Secure,
		HttpOnly: m.Options.HttpOnly,
	}
	session.IsNew = true
	var err error
	if cook, errToken := m.Token.GetToken(r, name); errToken == nil {
		err = securecookie.DecodeMulti(name, cook, &session.ID, m.Codecs...)
		if err == nil {
			err = m.load(session)
			if err == nil {
				session.IsNew = false
			} else {
				err = nil
			}
		}
	}
	return session, err
}

// Save saves all sessions registered for the current request.
func (m *MongoStore) Save(r *http.Request, w http.ResponseWriter,
	session *sessions.Session) error {
	if session.Options.MaxAge < 0 {
		if err := m.delete(session); err != nil {
			return err
		}
		m.Token.SetToken(w, session.Name(), "", session.Options)
		return nil
	}

	if session.ID == "" {
		session.ID = primitive.NewObjectID().Hex()
	}

	if err := m.upsert(session); err != nil {
		return err
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID,
		m.Codecs...)
	if err != nil {
		return err
	}

	m.Token.SetToken(w, session.Name(), encoded, session.Options)
	return nil
}

// MaxAge sets the maximum age for the store and the underlying cookie
// implementation. Individual sessions can be deleted by setting Options.MaxAge
// = -1 for that session.
func (m *MongoStore) MaxAge(age int) {
	m.Options.MaxAge = age

	// Set the maxAge for each securecookie instance.
	for _, codec := range m.Codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}

func (m *MongoStore) load(session *sessions.Session) error {
	oid, err := primitive.ObjectIDFromHex(session.ID)
	if err != nil {
		return errors.New("invalid session id")
	}

	var s Session
	err = m.coll.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&s)
	if err != nil {
		return err
	}

	if err := securecookie.DecodeMulti(session.Name(), s.Data, &session.Values,
		m.Codecs...); err != nil {
		return err
	}

	return nil
}

func (m *MongoStore) upsert(session *sessions.Session) error {
	oid, err := primitive.ObjectIDFromHex(session.ID)
	if err != nil {
		return errors.New("invalid session id")
	}

	var modified time.Time
	if val, ok := session.Values["modified"]; ok {
		modified, ok = val.(time.Time)
		if !ok {
			return errors.New("mongostore: invalid modified value")
		}
	} else {
		modified = time.Now()
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values, m.Codecs...)
	if err != nil {
		return err
	}

	s := Session{
		Id:       oid,
		Data:     encoded,
		Modified: modified,
	}

	_, err = m.coll.UpdateOne(
		context.Background(),
		bson.M{"_id": oid},
		bson.M{"$set": s},
		options.Update().SetUpsert(true))
	if err != nil {
		return err
	}

	return nil
}

func (m *MongoStore) delete(session *sessions.Session) error {
	oid, err := primitive.ObjectIDFromHex(session.ID)
	if err != nil {
		return errors.New("invalid session id")
	}

	_, err = m.coll.DeleteOne(context.Background(), bson.M{"_id": oid})
	if err != nil {
		return err
	}

	return nil
}
