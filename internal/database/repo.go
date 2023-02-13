package database

import (
	"bytes"
	"fmt"
	"time"

	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

// const (
// 	keyData    = "data"
// 	keyIndexes = "indexes"
// 	keyModify  = "modifyAt"
// )

func PersistData(id string, data []byte, database string, collection string, update bool) func(*bbolt.Tx) error {
	return func(tx *bbolt.Tx) error {
		bkdb, err := tx.CreateBucketIfNotExists([]byte(database))
		if err != nil {
			return err
		}
		bkcll, err := bkdb.CreateBucketIfNotExists([]byte(collection))
		if err != nil {
			return err
		}

		// logs.LogBuild.Printf("salida 0 id: %s", id)
		if len(id) <= 0 {
			if uid, err := uuid.NewUUID(); err != nil {
				return err
			} else {
				id = uid.String()
			}
		}
		if !update {
			if v := bkcll.Get([]byte(id)); v != nil {
				return ErrDataUpdateNotAllow
			}
		}
		if err := bkcll.Put([]byte(id), data); err != nil {
			return err
		}
		// if err := bkdata.Put([]byte(keyIndexes),
		// 	[]byte(fmt.Sprintf("[%s]", strings.Join(indexes, ",")))); err != nil {
		// 	return err
		// }
		// if err := bkdata.Put([]byte(keyModify), []byte(fmt.Sprintf("%d", time.Now().UnixNano()))); err != nil {
		// 	return err
		// }

		// for _, index := range indexes {
		// 	bkind, err := bkcll.CreateBucketIfNotExists([]byte(index))
		// 	if err != nil {
		// 		return err
		// 	}
		// 	bkind.Put([]byte(index), nil)
		// }

		return nil
	}
}

func GetData(data *[]byte, id, database, collection string) func(*bbolt.Tx) error {
	return func(tx *bbolt.Tx) error {
		bkdb := tx.Bucket([]byte(database))
		if bkdb == nil {
			logs.LogBuild.Println("ErrBucketNotFound")
			return nil
		}
		bkcll := bkdb.Bucket([]byte(collection))
		if bkcll == nil {
			logs.LogBuild.Println("ErrBucketNotFound")
			return nil
		}
		if len(id) <= 0 {
			return bbolt.ErrKeyRequired
		}
		value := bkcll.Get([]byte(id))
		if value == nil {
			logs.LogBuild.Println("DataNotFound")
			return nil
		}

		*data = append(*data, value...)

		return nil
	}
}

func RemoveData(id, database, collection string) func(*bbolt.Tx) error {
	return func(tx *bbolt.Tx) error {
		bkdb := tx.Bucket([]byte(database))
		if bkdb == nil {
			return bbolt.ErrBucketNotFound
		}
		bkcll := bkdb.Bucket([]byte(collection))
		if bkcll == nil {
			return bbolt.ErrBucketNotFound
		}

		if err := bkcll.Delete([]byte(id)); err != nil {
			return err
		}

		return nil
	}
}

type QueryType struct {
	Data       []byte
	Collection []byte
	ID         string
}

func SearchCollections(data *[]string, database string, prefixID []byte, reverse bool) func(*bbolt.Tx) error {
	return func(tx *bbolt.Tx) error {
		// log.Printf("database: %s, collection: %s, prefix: %s", database, collection, prefixID)
		bkdb := tx.Bucket([]byte(database))
		if bkdb == nil {
			return nil
		}

		c := bkdb.Cursor()
		if len(prefixID) > 0 {
			if reverse {
				c.Seek(prefixID)
				for k, _ := c.Last(); k != nil && bytes.HasPrefix(k, prefixID); k, _ = c.Prev() {
					// fmt.Printf("key=%s, value=%s\n", k, v)
					*data = append(*data, string(k))
				}
			} else {
				for k, _ := c.Seek(prefixID); k != nil && bytes.HasPrefix(k, prefixID); k, _ = c.Next() {
					// fmt.Printf("key=%s, value=%s\n", k, v)
					*data = append(*data, string(k))
				}
			}
		} else {
			if reverse {
				for k, _ := c.Last(); k != nil; k, _ = c.Prev() {
					// fmt.Printf("key=%s, value=%s\n", k, v)
					*data = append(*data, string(k))
				}
			} else {
				for k, _ := c.First(); k != nil; k, _ = c.Next() {
					// fmt.Printf("key=%s, value=%s\n", k, v)
					*data = append(*data, string(k))
				}
			}
		}
		return nil
	}
}

func QueryData(data chan *QueryType, stop chan int, database, collection string, prefixID []byte, reverse bool) func(*bbolt.Tx) error {
	return func(tx *bbolt.Tx) error {
		// log.Printf("database: %s, collection: %s, prefix: %s", database, collection, prefixID)
		defer close(data)
		bkdb := tx.Bucket([]byte(database))
		if bkdb == nil {
			return nil
		}
		bkcll := bkdb.Bucket([]byte(collection))
		if bkcll == nil {
			return nil
		}
		c := bkcll.Cursor()
		if len(prefixID) > 0 {
			if reverse {

				c.Seek(prefixID)
				for k, v := c.Last(); k != nil && bytes.HasPrefix(k, prefixID); k, v = c.Prev() {
					// fmt.Printf("key=%s, value=%s\n", k, v)
					replica := make([]byte, len(v))
					copy(replica, v)
					select {
					case <-stop:
						fmt.Printf("close query datadb")
						return nil
					case data <- &QueryType{ID: string(k), Data: replica}:
					case <-time.After(10 * time.Second):
						fmt.Printf("timeout query datadb, %s", collection)
					}
				}
			} else {
				for k, v := c.Seek(prefixID); k != nil && bytes.HasPrefix(k, prefixID); k, v = c.Next() {
					// fmt.Printf("key=%s, value=%s\n", k, v)
					replica := make([]byte, len(v))
					copy(replica, v)
					select {
					case <-stop:
						fmt.Printf("close query datadb")
						return nil
					case data <- &QueryType{ID: string(k), Data: replica}:
					case <-time.After(10 * time.Second):
						fmt.Printf("timeout query datadb, %s", collection)
					}
				}
			}
		} else {
			if reverse {
				for k, v := c.Last(); k != nil; k, v = c.Prev() {
					// fmt.Printf("key=%s, value=%s\n", k, v)
					replica := make([]byte, len(v))
					copy(replica, v)
					select {
					case <-stop:
						fmt.Printf("close query datadb")
						return nil
					case data <- &QueryType{ID: string(k), Data: replica}:
					case <-time.After(10 * time.Second):
						fmt.Printf("timeout query datadb, %s", collection)
					}
				}
			} else {
				for k, v := c.First(); k != nil; k, v = c.Next() {
					// fmt.Printf("key=%s, value=%s\n", k, v)
					replica := make([]byte, len(v))
					copy(replica, v)
					select {
					case <-stop:
						fmt.Printf("close query datadb")
						return nil
					case data <- &QueryType{ID: string(k), Data: replica}:
					case <-time.After(10 * time.Second):
						fmt.Printf("timeout query datadb, %s", collection)
					}
				}
			}
		}
		return nil
	}
}

func Last(data *[]byte, database, collection string, prefixID []byte) func(*bbolt.Tx) error {
	return func(tx *bbolt.Tx) error {
		bkdb := tx.Bucket([]byte(database))
		if bkdb == nil {
			return nil
		}
		bkcll := bkdb.Bucket([]byte(collection))
		if bkcll == nil {
			return nil
		}
		c := bkcll.Cursor()
		var k, v []byte
		if len(prefixID) > 0 {
			for k, v = c.Seek(prefixID); k != nil && bytes.HasPrefix(k, prefixID); k, v = c.Next() {
				// fmt.Printf("key=%s, value=%s\n", k, v)
			}
		} else {
			for k, v = c.First(); k != nil; k, v = c.Next() {
				// fmt.Printf("key=%s, value=%s\n", k, v)
			}
		}
		*data = make([]byte, len(v))
		copy(*data, v)
		return nil
	}
}
