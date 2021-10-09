package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

var client *mongo.Client

type User struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name  string             `json:"name,omitempty" bson:"name,omitempty"`
	Email string             `json:"email,omitempty" bson:"email,omitempty"`
	Password string          `json:"password,omitempty" bson:"password,omitempty"`
}

type Post struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Caption  string             `json:"caption,omitempty" bson:"caption,omitempty"`
	ImageURL string             `json:"imageurl,omitempty" bson:"imageurl,omitempty"`
	PostedTimestamp string      `json:"postedtimestamp,omitempty" bson:"postedtimestamp,omitempty"`
}

func createHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

func encrypt(data []byte, passphrase string) []byte {
	block, _ := aes.NewCipher([]byte(createHash(passphrase)))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext
}

func decrypt(data []byte, passphrase string) []byte {
	key := []byte(createHash(passphrase))
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err.Error())
	}
	return plaintext
}

func CreateUserEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("content-type", "application/json")
	var user User
	_ = json.NewDecoder(request.Body).Decode(&user)
	collection := client.Database("instagram-api").Collection("user")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	user.password=encrypt(user.password);
	result, _ := collection.InsertOne(ctx, user)
	json.NewEncoder(response).Encode(result)
}

func CreatePostEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("content-type", "application/json")
	var post Post
	_ = json.NewDecoder(request.Body).Decode(&post)
	app := fiber.New()
	app.Use(cors.New())

	app.Post("/api/post/populate", func(c *fiber.Ctx) error {
		collection := client.Database("instagram-api").Collection("post")
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
        
		for i := 0; i < 50; i++ {
			collection.InsertOne(ctx, Post{
				Caption:       faker.Word(),
				ImageURL:      fmt.Sprintf("http://lorempixel.com/200/200?%s", faker.UUIDDigit()),
				PostedTimestamp: faker.Time()      ,
			})
		}

		return c.JSON(fiber.Map{
			"message": "success",
		})
	})
	json.NewEncoder(response).Encode(result)
}

func GetUserEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("content-type", "application/json")
	params := mux.Vars(request)
	id, _ := primitive.ObjectIDFromHex(params["id"])
	var user User
	collection := client.Database("instagram-api").Collection("user")
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	user.password=decrypt(user.password)
	err := collection.FindOne(ctx, User{ID: id}).Decode(&user)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	json.NewEncoder(response).Encode(user)
}


func GetPostEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("content-type", "application/json")
	params := mux.Vars(request)
	id, _ := primitive.ObjectIDFromHex(params["id"])
	var post Post
	
	app.Get("/api/post/frontend", func(c *fiber.Ctx) error {
		collection := db.Collection("post")
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

		var products []Post

		cursor, _ := collection.Find(ctx, bson.M{})
		defer cursor.Close(ctx)

		for cursor.Next(ctx) {
			var product Post
			cursor.Decode(&post)
			posts = append(posts, post)
		}

		return c.JSON(posts)
	})

	app.Get("/api/posts/backend", func(c *fiber.Ctx) error {
		collection := db.Collection("posts")
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

		var products []Post

		filter := bson.M{}
		findOptions := options.Find()

		if s := c.Query("s"); s != "" {
			filter = bson.M{
				"$or": []bson.M{
					{
						"title": bson.M{
							"$regex": primitive.Regex{
								Pattern: s,
								Options: "i",
							},
						},
					},
					{
						"description": bson.M{
							"$regex": primitive.Regex{
								Pattern: s,
								Options: "i",
							},
						},
					},
				},
			}
		}

		page, _ := strconv.Atoi(c.Query("page", "1"))
		var perPage int64 = 9

		total, _ := collection.CountDocuments(ctx, filter)

		findOptions.SetSkip((int64(page) - 1) * perPage)
		findOptions.SetLimit(perPage)

		cursor, _ := collection.Find(ctx, filter, findOptions)
		defer cursor.Close(ctx)

		for cursor.Next(ctx) {
			var post Post
			cursor.Decode(&post)
			posts = append(posts, post)
		}

		return c.JSON(fiber.Map{
			"data":      posts,
			"total":     total,
			"page":      page,
			"last_page": math.Ceil(float64(total / perPage)),
		})
	})

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	json.NewEncoder(response).Encode(post)
}

func GetPostsEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("content-type", "application/json")
	var posts []Post
	collection := client.Database("instagram-api").Collection("posts")
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var post Post
		cursor.Decode(&post)
		posts = append(posts, post)
	}
	if err := cursor.Err(); err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	json.NewEncoder(response).Encode(posts)
}


func main() {
	fmt.Println("Starting the application...")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	clientOptions := options.Client().ApplyURI("mongodb+srv://shresth:shresth123@cluster0.wlqaa.mongodb.net/instagram-api?retryWrites=true&w=majority")
	client, _ = mongo.Connect(ctx, clientOptions)
	router := mux.NewRouter()
	
	router.HandleFunc("/user", CreateUserEndpoint).Methods("POST")
	router.HandleFunc("/user/{id}", GetUserEndpoint).Methods("GET")

	router.HandleFunc("/post", CreatePostEndpoint).Methods("POST")
	router.HandleFunc("/posts/user/{id}", GetPostsEndpoint).Methods("GET")
	router.HandleFunc("/post/{id}", GetPostEndpoint).Methods("GET")
	http.ListenAndServe(":12345", router)
}