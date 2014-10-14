package mongo

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	DbHost = "localhost"
	DbName = "chatbox"
)

var (
	session *mgo.Session
)

func connect() *mgo.Session {
	if session == nil {
		var err error
		session, err = mgo.Dial(DbHost)
		if err != nil {
			panic(err)
		}
	}
	return session.Clone()
}

func mongo(collection string, f func(*mgo.Collection)) {
	session := connect()
	defer func() {
		if err := recover(); err != nil {
			// Error("mongo", err)
		}
		session.Close()
	}()
	c := session.DB(DbName).C(collection)
	f(c)
}

func D(name string) MongoModel {
	m := MongoModel{Cname: name}
	return m
}

// type P bson.M //map[string]interface{}

type MongoModel struct {
	Cname string
	Query *bson.M // find/query condition
	Sk    int     // query start at
	L     int     // query max rows
	S     string  // sort
	F     *bson.M // select field
}

func (m MongoModel) Find(p ...bson.M) MongoModel {
	if len(p) == 1 {
		m.Query = &p[0]
	} else {
		m.Query = &bson.M{}
	}
	return m
}

func (m MongoModel) Field(s ...string) MongoModel {
	if m.F == nil {
		m.F = &bson.M{}
	}
	for _, k := range s {
		(*m.F)[k] = 1
	}
	return m
}

func (m MongoModel) Skip(start int) MongoModel {
	m.Sk = start
	return m
}

func (m MongoModel) Limit(rows int) MongoModel {
	m.L = rows
	return m
}

func (m MongoModel) Sort(s string) MongoModel {
	m.S = s
	return m
}

func (m MongoModel) All() *[]bson.M {
	ps := []bson.M{}
	mongo(m.Cname, func(c *mgo.Collection) {
		q := m.query(c)
		q.All(&ps)
	})
	return &ps
}

func (m MongoModel) One() *bson.M {
	p := bson.M{}
	mongo(m.Cname, func(c *mgo.Collection) {
		q := m.query(c)
		q.One(&p)
	})
	return &p
}

func (m MongoModel) Count() int {
	total := 0
	mongo(m.Cname, func(c *mgo.Collection) {
		q := m.query(c)
		total, _ = q.Count()
	})
	return total
}

func (m MongoModel) Add(docs ...interface{}) error {
	var err error
	mongo(m.Cname, func(c *mgo.Collection) {
		if len(docs) == 1 {
			err = c.Insert(docs[0])
		} else {
			err = c.Insert(docs)
		}
	})
	return err
}

func (m MongoModel) Save(p bson.M) error {
	var err error
	mongo(m.Cname, func(c *mgo.Collection) {
		id := p["_id"]
		var oid bson.ObjectId
		switch id.(type) {
		case string:
			oid = bson.ObjectIdHex(id.(string))
		case bson.ObjectId:
			oid = id.(bson.ObjectId)
		}
		p["_id"] = oid
		err = c.UpdateId(oid, p)
		if err != nil {
			// Error(err)
		}
	})
	return err
}

func (m MongoModel) RemoveId(id string) {
	mongo(m.Cname, func(c *mgo.Collection) {
		err := c.RemoveId(bson.ObjectIdHex(id))
		if err != nil {
			// Error(err)
		}
	})
}

func (m MongoModel) Remove(selector interface{}) {
	mongo(m.Cname, func(c *mgo.Collection) {
		err := c.Remove(selector)
		if err != nil {
			// Error(err)
		}
	})
}

func (m MongoModel) query(c *mgo.Collection) *mgo.Query {
	q := c.Find(m.Query).Skip(m.Sk)
	if m.L > 0 {
		q = q.Limit(m.L)
	}
	if len(m.S) > 0 {
		q = q.Sort(m.S)
	}
	if m.F != nil {
		q = q.Select(m.F)
	}
	return q
}
