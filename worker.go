package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"github.com/tidwall/gjson"
	"go.mongodb.org/mongo-driver/bson"
	"log"
	"net/http"
	"os"
	"rabbitMQService/config"
	"strings"
	"time"
)

type LinkPreviewResponse struct {
	Title string `bson:"title"`
	Description string `bson:"description"`
	Image string `bson:"image"`
	Url string `bson:"url"`
}

type Bookmark struct {
	Email  string		 `bson:"email"`
	Name       string        `bson:"name"`
	Link       string        `bson:"link"`
	Viewcount  int		 `bson:"viewcount"`
	Timestamp  string        `bson:"timestamp"`
	Image string `bson:"image"`
	Description string `bson:"description"`
	Category string `bson:"category"`
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func CreateBookmark(bookmark Bookmark) bool {
	var bookmarkWithSameUserLink Bookmark
	err := config.BookmarkCollection.FindOne(context.TODO(),
		bson.D{{"email", bookmark.Email},{Key: "link" , Value: bookmark.Link}}).Decode(&bookmarkWithSameUserLink)
	if err == nil {
		// A bookmark with same user email and link already exists.
		return false
	} else {
		// A bookmark with same user email and link does not exist.
		if _, err := config.BookmarkCollection.InsertOne(context.TODO(), bookmark); err != nil {
			return false
		} else {
			return true
		}
	}
}
func process(bookmark Bookmark) Bookmark{

	bookmark.Viewcount = 1
	dt := time.Now()
	bookmark.Timestamp = dt.String()

	if(!strings.HasPrefix(bookmark.Link, "http://") && !strings.HasPrefix(bookmark.Link, "https://")){
		bookmark.Link = "http://"+bookmark.Link
	}

	classifyurl := "https://www.klazify.com/api/categorize"
	payload := "{\"url\":\""+bookmark.Link+"\"}\n"
	linkClassifyAPIKey := os.Getenv("LINK_CLASSIFY_API_KEY")
	client := &http.Client{}

	req, err := http.NewRequest("POST", classifyurl, bytes.NewReader([]byte(payload)))

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+linkClassifyAPIKey)
	req.Header.Add("cache-control", "no-cache")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("couldn't find link category")
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	newStr := buf.String()
	value := gjson.Get(newStr, "domain.categories.0.name")
	s := strings.Split(value.String(), "/")
	bookmark.Category = s[len(s)-1]

	url := "http://api.linkpreview.net/?key=%s&q=%s"
	linkPreviewAPIKey := os.Getenv("LINK_PREVIEW_API_KEY")
	res, err := http.Get(fmt.Sprintf(url, linkPreviewAPIKey, bookmark.Link))
	if err != nil {
		log.Printf("couldn't find link preview")
	}
	defer res.Body.Close()
	linkPreview := &LinkPreviewResponse{}
	json.NewDecoder(res.Body).Decode(linkPreview)
	bookmark.Image = linkPreview.Image
	bookmark.Description = linkPreview.Description

	return bookmark
}
func receive(){
	//connect to RabbitMQ server
	//rabbitMQHost := os.Getenv("RABBITMQ_SERVICE_HOST")
	//conn, err := amqp.Dial(rabbitMQHost)
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	//create a channel
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	//declare a queue for us to consume
	q, err := ch.QueueDeclare(
		"task_queue", // name
		true,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var bookmark Bookmark
			json.Unmarshal(d.Body, &bookmark)
			bookmark = process(bookmark)
			CreateBookmark(bookmark)
			fmt.Println(bookmark)
			log.Printf("Received a message: %s", d.Body)
			log.Printf("Done")
			d.Ack(false)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
func main(){
	receive()
}
