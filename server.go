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

type Chapter struct {
	title  string  `json:"title"`
	quests []Quest `json:"quests"`
}

type Quest struct {
	title       string   `json:"title"`
	text        string   `json:"text"`
	regexps     []string `json:"regexps"`
	regexpsNone []string `json:"regexpsNone"`
	code        string   `json:"code"`
	hints       []string `json:"hints"`
	test        TestInfo `json:"test"`
}

type TestInfo struct {
	code   string `json:"code"`
	answer string `json:"answer"`
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
	http.Handle("/graphql", handler)
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
			"title": &graphql.Field{
				Type: graphql.String,
			},
			"quests": &graphql.Field{
				Type: graphql.NewList(QuestObject),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id := p.Args["id"]
					v, _ := id.(int)
					log.Printf("fetching quest with id: %d", v)
					return fetchQuestByiD(v)
				},
			},
		},
	})

	return graphql.ObjectConfig{Name: "chapters", Fields: graphql.Fields{
		"chapters": &graphql.Field{
			Type: graphql.NewList(ChapterObject),
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(graphql.Int),
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				id := p.Args["id"]
				v, _ := id.(int)
				log.Printf("fetching chapter with id: %d", v)
				return fetchChapterByiD(v)
			},
		},
	}}
}

func fetchChapterByiD(id int) (*Chapter, error) {
	query := "FOR d IN chapters FILTER d.Id == @id LIMIT 5 RETURN d"
	bindVars := map[string]interface{}{
		"id": id,
	}
	cursor, err := db.Query(nil, query, bindVars)
	if err != nil {
		log.Fatalf("Can not get quest: %v", err)
	}
	defer cursor.Close()
	result := Chapter{}
	for {
		var doc Chapter
		meta, err := cursor.ReadDocument(nil, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			log.Fatalf("Some issue(73): %v", err)
		}
		fmt.Printf("Got doc with key '%s' from query\n", meta)
		result = doc
	}
	return &result, nil
}

func fetchQuestByiD(id int) (*Quest, error) {
	query := "FOR d IN quests FILTER d.Id == @id LIMIT 5 RETURN d"
	bindVars := map[string]interface{}{
		"id": id,
	}
	cursor, err := db.Query(nil, query, bindVars)
	if err != nil {
		log.Fatalf("Can not get quest: %v", err)
	}
	defer cursor.Close()
	result := Quest{}
	for {
		var doc Quest
		meta, err := cursor.ReadDocument(nil, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			log.Fatalf("Some issue(73): %v", err)
		}
		fmt.Printf("Got doc with key '%s' from query\n", meta)
		result = doc
	}
	return &result, nil
}
