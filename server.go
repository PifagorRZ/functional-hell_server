package main

import (
	"fmt"
	"log"
	"net/http"

	"encoding/json"
	"errors"
	"io/ioutil"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"

	driver "github.com/arangodb/go-driver"
	driverHttp "github.com/arangodb/go-driver/http"
)

type Chapter {
	title string `json:"title"`
	quests Quest[] `json:"quests"`
}

type Quest {
	title string `json:"title"`
	text string `json:"text"`
	regexps string[] `json:"regexps"`
	regexpsNone string[] `json:"regexpsNone"`
	code string `json:"code"`
	hints string[] `json:"hints"`
	test TestInfo `json:"test"`
}

type TestInfo {
	code string `json:"code"`
	answer string `json:"answer"`
}


type Post struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// Comment is a comment
type Comment struct {
	PostID int    `json:"postId"`
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Body   string `json:"body"`
}

var db driver.Database

func main() {
	connectToDB()

	// col, err := db.Collection(nil, "quests")
	// if err != nil {
	// 	log.Fatalf("Can not connect to collection: %v", err)
	// }

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(
			createQueryType(
				createPostType(
					createCommentType(),
				),
			),
		),
	})
	if err != nil {
		log.Fatalf("failed to create new schema, error: %v", err)
	}
	handler := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/graphql", handler)
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/chat", chatHandler)
	http.HandleFunc("/quest", questHandler)
	http.HandleFunc("/", notFoundHandler)
	http.ListenAndServe(":80", nil)
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

func createQueryType(postType *graphql.Object) graphql.ObjectConfig {
	return graphql.ObjectConfig{Name: "QueryType", Fields: graphql.Fields{
		"post": &graphql.Field{
			Type: postType,
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(graphql.Int),
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				id := p.Args["id"]
				v, _ := id.(int)
				log.Printf("fetching post with id: %d", v)
				return fetchPostByiD(v)
			},
		},
	}}
}

func createPostType(commentType *graphql.Object) *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "Post",
		Fields: graphql.Fields{
			"userId": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
			},
			"id": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
			},
			"title": &graphql.Field{
				Type: graphql.String,
			},
			"body": &graphql.Field{
				Type: graphql.String,
			},
			"comments": &graphql.Field{
				Type: graphql.NewList(commentType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					post, _ := p.Source.(*Post)
					log.Printf("fetching comments of post with id: %d", post.ID)
					return fetchCommentsByPostID(post.ID)
				},
			},
		},
	})
}

func createCommentType() *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "Comment",
		Fields: graphql.Fields{
			"postId": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
			},
			"id": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
			},
			"name": &graphql.Field{
				Type: graphql.String,
			},
			"email": &graphql.Field{
				Type: graphql.String,
			},
			"body": &graphql.Field{
				Type: graphql.String,
			},
		},
	})
}

func fetchPostByiD(id int) (*Post, error) {
	query := "FOR d IN quests FILTER d.Id == @id LIMIT 5 RETURN d"
	bindVars := map[string]interface{}{
		"id": id,
	}
	cursor, err := db.Query(nil, query, bindVars)
	if err != nil {
		log.Fatalf("Can not get quests: %v", err)
	}
	defer cursor.Close()
	result := Post{}
	for {
		var doc Post
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

func fetchCommentsByPostID(id int) ([]Comment, error) {
	resp, err := http.Get(fmt.Sprintf("http://jsonplaceholder.typicode.com/posts/%d/comments", id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s: %s", "could not fetch data", resp.Status)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("could not read data")
	}
	result := []Comment{}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, errors.New("could not unmarshal data")
	}
	return result, nil
}
