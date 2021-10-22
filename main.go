package main

import (
	"context"
	"encoding/json"
	"net/http"
	"log"  
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/thedevsaddam/renderer"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//go get for packages
// go-chi /chi->

var rnd *renderer.Render
var db *mgo.Database


//mongo connections
const (
	hostName string ="localhost:27017"
	dbName string = "todo_db"
	collectionName string = "todo"
	port string = ":9000"
)


type(
	//for mongo model
	toDoModel struct{ 
		ID bson.ObjectId `bson:"_id,omitempty"` //Representation
		Title string `bson:"title"`
		Completed bool `bson:"completed"`
		CreatedAt time.Time  `bson:"createdAt"`
	}
	//json structure for code
	toDo struct
	{
		ID string `json:"id"`
		Title string `json:"title"`
		Completed bool `json:"completed"`
		CreatedAt time.Time `json:"createdAt"`
	}
)
 
//connect with db
func init(){
	rnd = renderer.New();
	sess,err := mgo.Dial(hostName)
	checkErr(err)
	sess.SetMode(mgo.Monotonic,true)
	db=sess.DB(dbName) //db
  }
  
func checkErr(err error){
	  if(err!=nil){
		  log.Fatal(err)
	  }
}


func todoHandler() http.Handler{
  rg := chi.NewRouter()
  rg.Group(func(r chi.Router){
	r.Get("/",fetchAllTodods)
	r.Post("/",postTodo)
	r.Put("/{id}",updateTodo)
	r.Delete("/{id}",deleteTodo)

  })
  return rg
}


func homeHandler(w http.ResponseWriter, r *http.Request){
	err:= rnd.Template(w,http.StatusOK,[]string{"static/home.tpl"},nil)
	checkErr(err)
}

func fetchAllTodods(w http.ResponseWriter, r *http.Request){
	todos := []toDoModel{} //slice of todo model
	//from db
	if err:= db.C(collectionName).Find(bson.M{}).All(&todos);err!=nil{

		rnd.JSON(w,http.StatusProcessing,renderer.M{
			"message":"Failed to fetch todos",
			"error":err,
		})
		return
	}
	//for frontend we will have to send json data  
	todoList:=[]toDo{}
	for _,t:=range(todos){
		todoList = append(todoList,toDo{ID:t.ID.Hex(),Title:t.Title,Completed: t.Completed,CreatedAt: t.CreatedAt})
	}
	rnd.JSON(w,http.StatusOK,renderer.M{
		"data":todoList,
	})
}

func postTodo(w http.ResponseWriter, r *http.Request){
	var  t toDo
	if err :=  json.NewDecoder(r.Body).Decode(&t);err!=nil{
		//in case of error
		rnd.JSON(w,http.StatusProcessing,err)
		return
	} //decode and put inside t
	
	//add t in the database
	if t.Title==""{
		rnd.JSON(w,http.StatusBadRequest,renderer.M{
			"message":"Title is required",
		})
		return
	}

	tm:=toDoModel{
		ID:bson.NewObjectId(),
		Title:t.Title,
		Completed:false,
		CreatedAt: time.Now(),
	}

	 
	if err:= db.C(collectionName).Insert(&tm);err!=nil{
		rnd.JSON(w,http.StatusProcessing,renderer.M{
			"message":"Failed to save todo",
			"error":err,
		})
		return
		
	}

	rnd.JSON(w,http.StatusCreated,renderer.M{
		"message":"Todo created successfully",
		"todId": tm.ID.Hex(),
	})

}

func updateTodo(w http.ResponseWriter, r *http.Request){
	id := strings.TrimSpace(chi.URLParam(r,"id"))
	//check id exists indb
	//bson in mongo
	if !bson.IsObjectIdHex(id){
		rnd.JSON(w,http.StatusBadRequest,renderer.M{
			"message":"Id is invalid",
		})
		return
	}

	var t toDo 
	//decoded and put into t
	if err:= json.NewDecoder(r.Body).Decode(&t);err!=nil{
		rnd.JSON(w,http.StatusProcessing,err)
		return
	}
	 if(t.Title==""){
		 rnd.JSON(w,http.StatusBadRequest,renderer.M{
			 "message":"Title field is required",
		 })
		 return; 
	 }

	 if err:=db.C(collectionName).Update(
		 bson.M{"_id":bson.ObjectIdHex(t.ID)},
		 bson.M{"title":t.Title,"completed":t.Completed},

	 );err!=nil{
		 rnd.JSON(w,http.StatusProcessing,renderer.M{
			 "message":"Failed to udpate",
			 "error":err,
		 })
		 return
	 }


}

func deleteTodo(w http.ResponseWriter, r *http.Request){
	id := strings.TrimSpace(chi.URLParam(r,"id"))
	//check id exists indb
	if !bson.IsObjectIdHex(id){
		rnd.JSON(w,http.StatusBadRequest,renderer.M{
			"message":"Id is invalid",
		})
		return
	}

	if err := db.C(collectionName).RemoveId(bson.ObjectIdHex(id));err!=nil{
		rnd.JSON(w,http.StatusProcessing,renderer.M{
			"message":"Failed to delete",
			"error":err,
		})
		return
	}

	rnd.JSON(w,http.StatusOK,renderer.M{
		"message":"Todo updated successfully",
	})
}



func main(){
	stopChan:= make(chan os.Signal)
	signal.Notify(stopChan,os.Interrupt)

	//create a router chi package
	r:= chi.NewRouter()
	//middleware
	r.Use(middleware.Logger);
	r.Get("/",homeHandler)
	r.Mount("/todo",todoHandler())

	srv := &http.Server{
		Addr:port,
		Handler:r,
		ReadTimeout : 60*time.Second,
		WriteTimeout : 60*time.Second,
		IdleTimeout : 60*time.Second,
	}
 
	go func(){
		log.Println("Listening on Port",port)
		if err:=srv.ListenAndServe();err!=nil{
			log.Printf("listen:%s\n",err)
		}
	}()

	<- stopChan
	log.Println("Shutting down server")
	ctx,cancel := context.WithTimeout(context.Background(),5*time.Second)
	srv.Shutdown(ctx)
	defer cancel()
	log.Println("Server stopped") 
}


