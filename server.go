package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"

	driver "github.com/arangodb/go-driver"
	driverHttp "github.com/arangodb/go-driver/http"
)

// Chapter model
type Chapter struct {
	Id     int     `json:"id"`
	Title  string  `json:"title"`
	Quests []Quest `json:"quests"`
}

// Quest info
type Quest struct {
	Id          int      `json:"id"`
	Title       string   `json:"title"`
	Text        string   `json:"text"`
	Regexps     []string `json:"regexps"`
	RegexpsNone []string `json:"regexpsNone"`
	Code        string   `json:"code"`
	Hints       []string `json:"hints"`
	Test        TestInfo `json:"test"`
}

// TestInfo model
type TestInfo struct {
	Code   string `json:"code"`
	Answer string `json:"answer"`
}

var db driver.Database

func main() {
	connectToDB()

	// col, err := db.Collection(nil, "quests")
	// if err != nil {
	//   log.Fatalf("Can not connect to collection: %v", err)
	// }

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(createQueryType()),
	})
	if err != nil {
		log.Fatalf("failed to create new schema, error: %v", err)
	}
	handler := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})

	http.HandleFunc("/ws", handleConnections)
	go handleMessages()
	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/graphql", disableCors(handler))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	// http.Handle("/chat", c.Handler(server))
	http.HandleFunc("/quest", questHandler)
	http.HandleFunc("/", notFoundHandler)
	http.ListenAndServe(":8080", nil)
}

func connectToDB() {
	conn, err := driverHttp.NewConnection(driverHttp.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529"},
	})
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	c, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication("root", "password"),
	})
	if err != nil {
		log.Fatalf("failed to create driver client for db: %v", err)
	}

	db_, err := c.Database(nil, "func_hell")
	if err != nil {
		log.Fatalf("Can not connect to database: %v", err)
	}

	db = db_
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "404 Not Found")
}

func createQueryType() graphql.ObjectConfig {
	var TestInfoObject = graphql.NewObject(graphql.ObjectConfig{
		Name: "test",
		Fields: graphql.Fields{
			"code": &graphql.Field{
				Type: graphql.String,
			},
			"answer": &graphql.Field{
				Type: graphql.String,
			},
		},
	})

	var QuestObject = graphql.NewObject(graphql.ObjectConfig{
		Name: "quest",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.ID,
			},
			"title": &graphql.Field{
				Type: graphql.String,
			},
			"text": &graphql.Field{
				Type: graphql.String,
			},
			"regexps": &graphql.Field{
				Type: graphql.NewList(graphql.String),
			},
			"regexpsNone": &graphql.Field{
				Type: graphql.NewList(graphql.String),
			},
			"code": &graphql.Field{
				Type: graphql.String,
			},
			"hints": &graphql.Field{
				Type: graphql.NewList(graphql.String),
			},
			"test": &graphql.Field{
				Type: TestInfoObject,
			},
		},
	})

	var ChapterObject = graphql.NewObject(graphql.ObjectConfig{
		Name: "chapter",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.ID,
			},
			"title": &graphql.Field{
				Type: graphql.String,
			},
			"quests": &graphql.Field{
				Type: graphql.NewList(QuestObject),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id := p.Args["id"]
					parentId := p.Source.(Chapter).Id
					if id != nil {
						v, _ := id.(int)
						log.Printf("fetching quest with id: %d", v)
						return fetchQuestByiD(parentId, v)
					} else {
						log.Printf("fetching all quests")
						return fetchQuestsByChapterId(parentId)
					}
				},
			},
		},
	})

	return graphql.ObjectConfig{Name: "chapters", Fields: graphql.Fields{
		"chapters": &graphql.Field{
			Type: graphql.NewList(ChapterObject),
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{
					Type: graphql.Int,
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				id := p.Args["id"]
				if id != nil {
					v, _ := id.(int)
					log.Printf("fetching chapter with id: %d", v)
					return fetchChapterByiD(v)
				} else {
					log.Printf("fetching all chapters")
					return fetchAllChapters()
				}
			},
		},
	}}
}

func fetchAllChapters() (*[]Chapter, error) {
	query := "FOR c IN quests RETURN c"
	cursor, err := db.Query(nil, query, nil)
	if err != nil {
		log.Fatalf("Can not get all chapters: %v", err)
	}
	defer cursor.Close()
	var result []Chapter
	for {
		var doc Chapter
		meta, err := cursor.ReadDocument(nil, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			log.Fatalf("Some issue(73): %v", err)
		}
		fmt.Printf("Got doc with key '%s' from query\n", meta)
		result = append(result, doc)
	}
	return &result, nil
}

func fetchQuestsByChapterId(parentId int) (*[]Quest, error) {
	query := "FOR c IN quests FILTER c.id == @chapterId FOR q IN c.quests RETURN q"
	bindVars := map[string]interface{}{
		"chapterId": parentId,
	}
	cursor, err := db.Query(nil, query, bindVars)
	if err != nil {
		log.Fatalf("Can not get quest: %v", err)
	}
	defer cursor.Close()
	var result []Quest
	for {
		var doc Quest
		meta, err := cursor.ReadDocument(nil, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			log.Fatalf("Some issue(73): %v", err)
		}
		fmt.Printf("Got doc with key '%s' from query\n", meta)
		result = append(result, doc)
	}
	return &result, nil
}

func fetchChapterByiD(id int) (*[]Chapter, error) {
	query := "FOR d IN quests FILTER d.id == @chapterId RETURN d"
	bindVars := map[string]interface{}{
		"chapterId": id,
	}
	cursor, err := db.Query(nil, query, bindVars)
	if err != nil {
		log.Fatalf("Can not get chapter: %v", err)
	}
	defer cursor.Close()
	var result []Chapter
	for {
		var doc Chapter
		meta, err := cursor.ReadDocument(nil, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			log.Fatalf("Some issue(73): %v", err)
		}
		fmt.Printf("Got doc with key '%s' from query\n", meta)
		result = append(result, doc)
	}
	return &result, nil
}

func fetchQuestByiD(parentId int, id int) (*[]Quest, error) {
	query := "FOR c IN quests FILTER c.id == @chapterId FOR q IN c.quests FILTER q.id == @questId RETURN q"
	bindVars := map[string]interface{}{
		"questId":   id,
		"chapterId": parentId,
	}
	cursor, err := db.Query(nil, query, bindVars)
	if err != nil {
		log.Fatalf("Can not get quest: %v", err)
	}
	defer cursor.Close()
	var result []Quest
	for {
		var doc Quest
		meta, err := cursor.ReadDocument(nil, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			log.Fatalf("Some issue(73): %v", err)
		}
		fmt.Printf("Got doc with key '%s' from query\n", meta)
		result = append(result, doc)
	}
	return &result, nil
}

func disableCors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Content-Length, Accept-Encoding")
	if r.Method == "OPTIONS" {
	  w.Header().Set("Access-Control-Max-Age", "86400")
	  w.WriteHeader(http.StatusOK)
	  return
	}
	h.ServeHTTP(w, r)
   })
}
