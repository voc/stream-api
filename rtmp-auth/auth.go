package main

import (
    // "os"
    "fmt"
    "log"
    "flag"
    "bytes"
    "strconv"
    "io"
    "sync"
    "io/ioutil"
    "net/http"
    "crypto/md5"
    "crypto/rand"
    "text/template"
    "github.com/gorilla/mux"
    "github.com/gorilla/schema"
    "github.com/golang/protobuf/proto"
    "bitbucket.fem.tu-ilmenau.de/scm/~ischluff/stream-api/storage"
)

type handleFunc func(http.ResponseWriter, *http.Request)

type Store struct {
    state storage.State
    sync.RWMutex
}

func (store *Store) Find(app, name, auth) {

}

    // call=play
    // addr - client IP address
    // clientid - nginx client id (displayed in log and stat)
    // app - application name
    // flashVer - client flash version
    // swfUrl - client swf url
    // tcUrl - tcUrl
    // pageUrl - client page url
    // name - stream name

type Publish struct {
    Call string
    Addr string
    Clientid string
    App string
    FlashVer string
    Name string
    PageUrl string
    SwfUrl string
    TcUrl string
    Type string
    Auth string
}

var decoder = schema.NewDecoder()
var store Store


func genKey() {
    c := 10
    b := make([]byte, c)
    _, err := rand.Read(b)
    if err != nil {
        fmt.Println("error:", err)
        return
    }
    // The slice should now contain random bytes instead of only zeroes.
    fmt.Println(bytes.Equal(b, make([]byte, c)))
}

func genToken() string {
    h := md5.New()
    io.WriteString(h, strconv.FormatInt(123456, 10))
    io.WriteString(h, "ganraomaxxxxxxxxx")
    return fmt.Sprintf("%x", h.Sum(nil))
}

func PublishHandler() handleFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        err := r.ParseForm()
        if err != nil {
            log.Println("Invalid publish:", err)
            http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
            return
        }

        var publish Publish

        // r.PostForm is a map of our POST form values
        err = decoder.Decode(&publish, r.PostForm)
        if err != nil {
            log.Println("Invalid publish:", err)
            http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
            return
        }

        // w.WriteHeader(http.StatusOK)
        log.Println("publish", publish)
    }

    // call=play
    // addr - client IP address
    // clientid - nginx client id (displayed in log and stat)
    // app - application name
    // flashVer - client flash version
    // swfUrl - client swf url
    // tcUrl - tcUrl
    // pageUrl - client page url
    // name - stream name



    // attention: If you do not call ParseForm method, the following data can not be obtained form
    // fmt.Println(r.Form) // print information on server side.
    // fmt.Println("path", r.URL.Path)
    // fmt.Fprintf(w, "Hello astaxie!") // write data to response

    // stream, err := Lookup(vars.Name)
    // if err != nil {
    //     http.Error(w, err.Error(), http.StatusForbidden)
    //     return
    // }

    // stream.active = true
    // return


    // vars.Name
    // w.WriteHeader(http.StatusOK)
    // fmt.Fprintf(w, "Category: %v\n", vars["category"])

    // for _, auth := {
    //     range
    // }
}

func UnpublishHandler() handleFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // vars := mux.Vars(r)
        // w.WriteHeader(http.StatusOK)
        r.ParseForm()
        log.Println("unpublish", r.Form)
    }
}


func Template(name string, w http.ResponseWriter) {
    p := Page{Title: "Heading"}
    err := templates.ExecuteTemplate(w, name + ".html", p)
    if err != nil {
        log.Fatal("Cannot Get View ", err)
    }
}

func ReadState(path string) (*storage.State) {
    data, err := ioutil.ReadFile(path)
    if err != nil {
        log.Println("No previous file read", err)
        return nil
    }
    state := &storage.State{}
    if err := proto.Unmarshal(data, state); err != nil {
        log.Fatalln("Failed to parse stream state", err)
    }

    fmt.Printf("File contents: %s", state)
    return state
}

func StoreState(path *string, state storage.State) (error) {
    return nil
}

type Page struct {
    Title string
}

var templates = template.Must(template.ParseGlob("templates/*"))

func main() {
    var path = flag.String("store", "store.db", "store file")
    flag.Parse()

    previous := ReadState(*path);
    if previous != nil {
        store.state = *previous;
    }

    router := mux.NewRouter()
    router.HandleFunc("/publish", PublishHandler()).Methods("POST")
    router.HandleFunc("/unpublish", UnpublishHandler()).Methods("POST")
    router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        Template("form", w)
        // http.Error(w, "404 Not Found", http.StatusNotFound)
    })
    // http.Handle("/", router)

    log.Println("foobar")

    log.Fatal(http.ListenAndServe(":8080", router))

    // read auth
    // dat, err := ioutil.ReadFile("auth.db")
    // check(err)
    // fmt.Print(string(dat))

    // d2 := []byte{115, 111, 109, 101, 10}
    // n2, err := f.Write(d2)
    // check(err)
    // fmt.Printf("wrote %d bytes\n", n2)




    // n3, err := f.WriteString("writes\n")
    // fmt.Printf("wrote %d bytes\n", n3)

    // f.Sync()
}
