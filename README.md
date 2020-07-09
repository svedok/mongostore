mongostore
==========

[Gorilla's Session](http://www.gorillatoolkit.org/pkg/sessions) store implementation with MongoDB

## Installation

    go get github.com/svedok/mongostore

### Example
```go
    func foo(rw http.ResponseWriter, req *http.Request) {
        // Fetch new store.
        dbsess, err := mgo.Dial("localhost")
        if err != nil {
            panic(err)
        }
        defer dbsess.Close()

        store := mongostore.NewMongoStore(dbsess.DB("test").C("test_session"), 3600, true,
            []byte("secret-key"))

        // Get a session.
        session, err := store.Get(req, "session-key")
        if err != nil {
            log.Println(err.Error())
        }

        // Add a value.
        session.Values["foo"] = "bar"

        // Save.
        if err = sessions.Save(req, rw); err != nil {
            log.Printf("Error saving session: %v", err)
        }

        fmt.Fprintln(rw, "ok")
    }
```
