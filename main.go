package main

import (
	"context"
	"encoding/json"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"fmt"
	models "github.com/horcu/peez_me_models"
	"html/template"
	"log"
	"net/http"
	"os"
)

// Variables used to generate the HTML page.
var (
	data       models.TemplateData
	tmpl       *template.Template
	ticketsRef *db.Ref
	userRef    *db.Ref
	invitesRef *db.Ref
)

// Simple init to have a client, etc... available
func setup() {

	// set context
	_ = context.Background()

	// global variables here like firebase client etc...
	conf := &firebase.Config{
		ProjectID:   "peezme",
		DatabaseURL: "https://peezme-default-rtdb.firebaseio.com/",
	}

	// new firebase app instance
	app, err := firebase.NewApp(context.Background(), conf)
	if err != nil {
		_ = fmt.Errorf("error initializing firebase database app: %v", err)
	}

	// connect to the database
	database, err := app.Database(context.Background())

	if err != nil {
		_ = fmt.Errorf("error connecting to the database app: %v", err)
	}

	// Get a database reference to our game details.
	ticketsRef = database.NewRef("requests/")
	userRef = database.NewRef("users/")
	invitesRef = database.NewRef("invitations/")

}

func main() {
	setup()

	// Initialize template parameters.
	service := os.Getenv("K_SERVICE")
	if service == "" {
		service = "???"
	}

	revision := os.Getenv("K_REVISION")
	if revision == "" {
		revision = "???"
	}

	// Prepare template for execution.
	tmpl = template.Must(template.ParseFiles("index.html"))
	data = models.TemplateData{
		Service:  service,
		Revision: revision,
	}

	// Define HTTP server.
	mux := http.NewServeMux()
	//mux.HandleFunc("/", invitationHandler)
	mux.HandleFunc("/invite", invitationHandler)

	fs := http.FileServer(http.Dir("./assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", fs))

	// PORT environment variable is provided by Cloud Run.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Print("Hello from Cloud Run! The container started successfully and is listening for HTTP requests on $PORT")
	log.Printf("Listening on port %s", port)
	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		log.Fatal(err)
	}
}

// helloRunHandler responds to requests by rendering an HTML page.
func invitationHandler(w http.ResponseWriter, r *http.Request) {
	//if err := tmpl.Execute(w, data); err != nil {
	//	msg := http.StatusText(http.StatusInternalServerError)
	//	log.Printf("template.Execute: %v", err)
	//	http.Error(w, msg, http.StatusInternalServerError)
	//}

	var tick *models.Ticket
	if err := json.NewDecoder(r.Body).Decode(&tick); err != nil {
		fmt.Println("error: " + err.Error())
		returnAccepted(w)
	}

	fmt.Fprintln(w, "ticket parsed")

	// ensure that this is an invitation request
	if tick.IsMatchTicket {
		msg := "incorrect ticket type"
		fmt.Println("error: " + msg)
		returnAccepted(w)
	}

	fmt.Fprintln(w, "finding users")

	for i, invitee := range tick.Invitees {
		u := models.User{}
		err := userRef.Child(invitee.ID).Get(context.Background(), &u)
		if err != nil {
			_, err = fmt.Println("could not find user with Id: " + invitee.ID)
			continue
		}
		tick.Invitees[i] = u
	}

	//save the ticket record
	ref, err := ticketsRef.Child(tick.CreatedBy).Push(context.Background(), nil)
	if err != nil {
		fmt.Println("error: " + err.Error())
		returnAccepted(w)
	}

	k := ref.Key
	tick.Id = k

	if ticketsRef.Child(tick.CreatedBy).Child(tick.Id).Set(context.Background(), &tick); err != nil {
		fmt.Println("error: " + err.Error())
		returnAccepted(w)
	}

	fmt.Println("tickets saved!")

	// create an invitation record for each user
	for _, inv := range tick.Invitees {
		err = invitesRef.Child(inv.ID).Child(tick.Id).Set(context.Background(), &tick)
		if err != nil {
			fmt.Fprintln(w, "error: "+err.Error())
			returnAccepted(w)
		}
	}

	_, err = fmt.Println("invitation saved!")
}

func returnAccepted(w http.ResponseWriter) {
	w.WriteHeader(http.StatusAccepted)
	return
}
