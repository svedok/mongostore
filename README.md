mongostore
==========

MongoDB implementation for [Gorilla's Session](http://www.gorillatoolkit.org/pkg/sessions) store using the official [MongoDB](https://github.com/mongodb/mongo-go-driver) driver for Go.

## Installation

    go get github.com/svedok/mongostore

### Example
```go
    func foo(rw http.ResponseWriter, req *http.Request) {
        // Fetch new store.
        m, err := mongo.Connect(context.Background())
        if err != nil {
            panic(err)
        }
        defer m.Disconnect(context.Background())

        store, err := NewMongoStore(m.Database("test").Collection("test_session"), 3600, true, []byte("secret-key"))
        if err != nil {
            panic(err)
        }

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
