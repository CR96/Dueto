package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/badoux/checkmail"
	"golang.org/x/crypto/bcrypt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"
)

const (
	AddArtist  = "insert into Artist(username, name, age, email, password, followers, description, likeCount, location, date, active) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9::json, now()::timestamp, true);"
	AddVideo   = "insert into Video(artistId, title, description, uploadTime, views, likes, genre, shared) VALUES($1, $2, $3, now()::timestamp, 0, 0, $4, false);"
	ShareVideo = "insert into Video(artistId, title, description, uploadTime, views, likes, genre, shared) VALUES($1, $2, $3, now()::timestamp, 0, 0, $4, true);"
	AddGenre   = "insert into Genre(name, description) VALUES($1, $2);"
	AddSession = "insert into Session(userId, sessionKey, time) VALUES($1, $2, now()::timestamp);"
	AddMessage = "insert into Comment(sender, reciever, message, time, sent) VALUES($1, $2, $3, now()::timestamp, false);"

	RemoveSession     = "delete from Session where sessionkey = $1;"
	RemoveOldSessions = "delete from session where age(now(), time) > '1 hour';"

	SelectNewVideoId      = "select id from video where artistId = $1 and title = $2 order by uploadtime desc limit 1;"
	SelectSharedVideos    = "select id, title, description, artistId, uploadTime, views, likes, genre from Video where artistId = $1 and shared = true;"
	SelectBasicArtistData = "select id, username, name from artist where id = $1;"
	SelectIntArtistData   = "select username, name, followers, description, date, active, likeCount, email, location::json->>'country', location::json->>'city', location::json->>'zip' from Artist where id = $1;"
	SelectExtArtistData   = "select username, name, description, date, active, followerCount, likeCount from Artist where id = $1;"
	SelectArtistVideos    = "select id, title, description, artistId, uploadTime, views, likes, genre from Video where artistId = $1 and shared = false;"
	SelectRecentVideos    = "select id, title, description, artistId, uploadTime, views, likes, genre from Video order by uploadtime desc limit 10;"
	SelectVideosByArtist  = "select id, title, description, views, likes, uploadTime, genre from Video where artistId = $1 and shared = false;"
	SelectVideosByGenre   = "select id, title, description, views, likes, uploadTime, artistId from Video where genre = $1;"
	SelectVideoById       = "select id, title, description, artistId, uploadTime, views, likes, genre from Video where id = $1"
	SelectGenres          = "select name, description from Genre;"
	SelectUserAuth        = "select id, password from artist where username = $1;"
	SelectSession         = "select count(userId) from session where sessionkey = $1;"
	SelectAuthId          = "select userId from session where sessionKey = $1;"
	SelectArtistByZip     = "select id, name, username from artist where location::json->>'zip'::text = (select location::json->>'zip' from artist where id = $1)::text and id != $1;"
	SelectArtistByCity    = "select id, name, username from artist where location::json->>'city'::text = (select location::json->>'city' from artist where id = $1)::text and id != $1;"
	SelectVideoLoc        = "select artistId, title from video where id = $1;"
	SelectUnreadM         = "select id, sender, reciever, message, time from Comment where (sender = $1 or sender= $2) and (reciever = $1 or reciever = $2) and sent = false;"
	SelectChatThread      = "select id, sender, reciever, message, time from Comment where (CAST(sender as varchar(8)) = $1 or CAST(sender as varchar(8)) = $2) and (CAST(reciever as varchar(8)) = $1 or CAST(reciever as varchar(8000)) = $2);"

	UpdateSent        = "update Comment set sent = true where id = $1;"
	UpdateArtist      = "update artist set username = $1, name = $2, description = $3, password = $4, email = $5 where id = $6;"
	UpdateBasicArtist = "update artist set username = $1, name = $2, description = $3, email = $4 where id = $5;"
	IncreaseViews     = "update video set views = (select views from video where id = $1) + 1 where id = $1;"
)

type Authentication struct {
	Username string
	Password string
}

type Message struct {
	Artist   string
	Receiver string
	Message  string
	Time     string
}

type PostMessage struct {
	Artist  string
	Message string
}

type NewVideo struct {
	Video string
	Name  string
	Desc  string
}

type NewUser struct {
	Name       string
	Username   string
	Password   string
	Repassword string
	Age        string
	Loc        string
	Bio        string
	Email      string
}

type Genre struct {
	Name        string
	Description string
}

type Genres struct {
	GenreList []Genre
}

type BasicArtist struct {
	Id       string
	Name     string
	Username string
}

type Video struct {
	Artist BasicArtist
	Id     string
	File   string
	Title  string
	Desc   string
	Tags   string
	Genre  string
	Likes  string
	Views  string
	Time   string
}

type VideoList struct {
	VideoCards []Video
}

type ExtArtist struct {
	Username      string
	Name          string
	Age           string
	Active        string
	Desc          string
	Date          string
	FollowerCount string
	LikeCount     string
	Country       string
	City          string
	Zipcode       string
	VideoList     []Video
}

type IntArtist struct {
	Username  string
	Name      string
	Followers string
	Desc      string
	Date      string
	Active    string
	LikeCount string
	Email     string
	Id        string
	Country   string
	City      string
	Zipcode   string
	VideoList []Video
}

type Comment struct {
	Id      string
	artist  BasicArtist
	Message string
	Time    string
}

func (data *IntArtist) SetVideoList(videos []Video) {
	data.VideoList = videos
}

func query(sql string) {
	_, err := db.Query(sql)
	logIfErr(err)
}

func authenticate(cookie *http.Cookie) bool {
	var sessionCount int

	if cookie.String() != "" {
		sessionId := cookie.Value

		rows, err := db.Query(SelectSession, sessionId)
		checkErr(err)

		rows.Next()
		err = rows.Scan(&sessionCount)
		logIfErr(err)
		rows.Close()

		if sessionCount > 0 {
			return true
		}
	}

	return false
}

func getUserId(sessionId string) string {
	var id string
	rows, err := db.Query(SelectAuthId, sessionId)
	logIfErr(err)

	rows.Next()
	err = rows.Scan(&id)
	logIfErr(err)
	rows.Close()

	return id
}

func artist(w http.ResponseWriter, r *http.Request) {
	var v Video
	var a ExtArtist

	w.Header().Set("Access-Control-Allow-Origin", "*")
	cookie, _ := r.Cookie("SESSIONID")
	artistId := r.URL.Query().Get("artist")

	if !authenticate(cookie) {
		http.Error(w, "Authentication failed", http.StatusForbidden)
		return
	}

	rows, err := db.Query(SelectExtArtistData, artistId)
	checkErr(err)

	rows.Next()
	err = rows.Scan(&a.Username, &a.Name, &a.Desc, &a.Date, &a.Active, &a.FollowerCount, &a.LikeCount, &a.Country, &a.City, &a.Zipcode)
	logIfErr(err)
	rows.Close()

	videoRows, viderr := db.Query(SelectArtistVideos, artistId)
	logIfErr(viderr)

	for videoRows.Next() {
		err = videoRows.Scan(&v.Id, &v.Title, &v.Desc, &artistId, &v.Time, &v.Views, &v.Likes, &v.Genre)
		logIfErr(err)

		a.VideoList = append(a.VideoList, v)
	}
	videoRows.Close()

	err = json.NewEncoder(w).Encode(a)
	logIfErr(err)
}

func profile(w http.ResponseWriter, r *http.Request) {
	var a IntArtist
	var videos []Video

	w.Header().Set("Access-Control-Allow-Origin", "*")
	cookie, _ := r.Cookie("SESSIONID")

	if !authenticate(cookie) {
		http.Error(w, "Authentication failed", http.StatusForbidden)
		return
	}

	artistId := getUserId(cookie.Value)
	a.Id = artistId

	rows, err := db.Query(SelectIntArtistData, artistId)
	checkErr(err)

	rows.Next()
	err = rows.Scan(&a.Username, &a.Name, &a.Followers, &a.Desc, &a.Date, &a.Active, &a.LikeCount, &a.Email, &a.Country, &a.City, &a.Zipcode)
	logIfErr(err)
	rows.Close()

	videoRows, viderr := db.Query(SelectArtistVideos, artistId)
	logIfErr(viderr)

	for videoRows.Next() {
		var v Video
		err = videoRows.Scan(&v.Id, &v.Title, &v.Desc, &artistId, &v.Time, &v.Views, &v.Likes, &v.Genre)
		logIfErr(err)

		videos = append(videos, v)
	}
	videoRows.Close()

	a.SetVideoList(videos)

	err = json.NewEncoder(w).Encode(a)
	logServerErr(w, err)
}

func homePage(w http.ResponseWriter, r *http.Request) {
	var videos VideoList

	w.Header().Set("Access-Control-Allow-Origin", "*")
	cookie, err := r.Cookie("SESSIONID")

	if !authenticate(cookie) {
		http.Error(w, "Authentication failed", http.StatusForbidden)
		return
	}

	rows, err := db.Query(SelectRecentVideos)
	logIfErr(err)

	for rows.Next() {
		var v Video
		var artistId string

		err = rows.Scan(&v.Id, &v.Title, &v.Desc, &artistId, &v.Time, &v.Views, &v.Likes, &v.Genre)
		logIfErr(err)

		var a BasicArtist
		artistRow, aErr := db.Query(SelectBasicArtistData, artistId)
		logIfErr(aErr)

		artistRow.Next()
		err = artistRow.Scan(&a.Id, &a.Username, &a.Name)
		logIfErr(err)
		artistRow.Close()

		v.Artist = a

		videos.VideoCards = append(videos.VideoCards, v)
	}
	rows.Close()

	err = json.NewEncoder(w).Encode(videos)
	logServerErr(w, err)
}

func discover(w http.ResponseWriter, r *http.Request) {
	var g Genre
	var genres Genres

	w.Header().Set("Access-Control-Allow-Origin", "*")
	cookie, err := r.Cookie("SESSIONID")

	if !authenticate(cookie) {
		http.Error(w, "Authentication failed", http.StatusForbidden)
		return
	}

	rows, err := db.Query(SelectGenres)
	checkErr(err)

	for rows.Next() {
		err := rows.Scan(&g.Name, &g.Description)
		logIfErr(err)

		genres.GenreList = append(genres.GenreList, g)
	}
	rows.Close()

	err = json.NewEncoder(w).Encode(genres)
	logServerErr(w, err)
}

func genre(w http.ResponseWriter, r *http.Request) {
	var videos VideoList

	w.Header().Set("Access-Control-Allow-Origin", "*")
	cookie, err := r.Cookie("SESSIONID")

	if !authenticate(cookie) {
		http.Error(w, "Authentication failed", http.StatusForbidden)
		return
	}
	genre := r.URL.Query().Get("genre")

	rows, err := db.Query(SelectVideosByGenre, genre)
	logIfErr(err)

	for rows.Next() {
		var v Video
		var artistId string
		var a BasicArtist

		err = rows.Scan(&v.Id, &v.Title, &v.Desc, &v.Views, &v.Likes, &v.Time, &artistId)
		logIfErr(err)

		v.Genre = genre
		artistRow, err := db.Query(SelectBasicArtistData, artistId)
		logServerErr(w, err)

		if artistRow.Next() {
			err = artistRow.Scan(&a.Id, &a.Username, &a.Name)
		}

		logServerErr(w, err)
		artistRow.Close()

		a.Id = artistId
		v.Artist = a

		videos.VideoCards = append(videos.VideoCards, v)
	}
	rows.Close()

	err = json.NewEncoder(w).Encode(videos)
	logServerErr(w, err)
}

func fileLoc(w http.ResponseWriter, id string) (artist string, video string) {
	var artistId string
	var videoName string

	rows, err := db.Query(SelectVideoLoc, id)
	logIfErr(err)

	if rows.Next() {
		rows.Scan(&artistId, &videoName)
	} else {
		http.Error(w, "Video not found", http.StatusNotFound)
		return "", ""
	}
	rows.Close()

	return artistId, videoName
}

func video(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	videoId := r.URL.Query().Get("id")

	artistId, _ := fileLoc(w, videoId)
	filePath := "./data/videos/" + artistId + "/" + videoId + ".mp4"

	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		http.Error(w, "Video not found", http.StatusNotFound)
		return
	}

	rows, err := db.Query(IncreaseViews, videoId)
	logIfErr(err)
	rows.Close()

	http.ServeFile(w, r, filePath)
}

func avatar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	artistId := r.URL.Query().Get("artist")

	filePath := "./data/avatars/" + artistId + "/avatar.jpeg"
	_, err := os.Stat(filePath)

	if os.IsNotExist(err) {
		http.Error(w, "Image not found", http.StatusNotFound)
	}

	http.ServeFile(w, r, filePath)
}

func thumbnail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	videoId := r.URL.Query().Get("id")

	artistId, _ := fileLoc(w, videoId)
	filePath := "./data/thumbnails/" + artistId + "/" + videoId + ".jpeg"
	_, err := os.Stat(filePath)

	if os.IsNotExist(err) {
		http.Error(w, "Image not found", http.StatusNotFound)
	}

	http.ServeFile(w, r, filePath)
}

func genreImage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	genreName := r.URL.Query().Get("name")

	filePath := "./data/genres/" + genreName + ".png"
	http.ServeFile(w, r, filePath)
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var id string
	var hashPassword string
	var data NewUser

	w.Header().Set("Access-Control-Allow-Origin", "*")
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(&data)

	if data.Password != data.Repassword {
		http.Error(w, "Passwords do not match", http.StatusNotAcceptable)
		return
	}

	err := checkmail.ValidateFormat(data.Email)
	if err != nil {
		http.Error(w, "Enter a valid email", http.StatusNotAcceptable)
	}

	bHash, err := bcrypt.GenerateFromPassword([]byte(data.Password), 1)
	hash := string(bHash)

	rows, err := db.Query(AddArtist, data.Username, data.Name, data.Age, data.Email, hash, 0, data.Bio, 0, data.Loc)
	logIfErr(err)
	rows.Close()

	authRows, err := db.Query(SelectUserAuth, data.Username)
	logServerErr(w, err)

	if authRows.Next() {
		err = authRows.Scan(&id, &hashPassword)
		logIfErr(err)
	} else {
		http.Error(w, "Unable to create user", http.StatusForbidden)
	}
	authRows.Close()

	os.Mkdir("./data/videos/"+id, os.ModePerm)
	os.Mkdir("./data/thumbnails/"+id, os.ModePerm)
	os.Mkdir("./data/avatars/"+id, os.ModePerm)
}

func editprofile(w http.ResponseWriter, r *http.Request) {
	var id string
	var hashPassword string
	var data NewUser

	w.Header().Set("Access-Control-Allow-Origin", "*")
	cookie, _ := r.Cookie("SESSIONID")

	if !authenticate(cookie) {
		http.Error(w, "Authentication failed", http.StatusForbidden)
		return
	}

	artist := getUserId(cookie.Value)

	decoder := json.NewDecoder(r.Body)
	decoder.Decode(&data)

	if data.Password == "" {
		rows, err := db.Query(UpdateBasicArtist, data.Username, data.Name, data.Bio, data.Email, artist)
		logIfErr(err)
		rows.Close()

		return
	}

	if data.Password != data.Repassword {
		http.Error(w, "Passwords do not match", http.StatusNotAcceptable)
		return
	}

	bHash, err := bcrypt.GenerateFromPassword([]byte(data.Password), 1)
	logServerErr(w, err)
	hash := string(bHash)

	rows, err := db.Query(UpdateArtist, data.Username, data.Name, data.Bio, hash, data.Email, artist)
	logIfErr(err)
	rows.Close()

	authRows, err := db.Query(SelectUserAuth, data.Username)
	logIfErr(err)

	if authRows.Next() {
		err = authRows.Scan(&id, &hashPassword)
		logIfErr(err)
	} else {
		http.Error(w, "Unable to create user", http.StatusForbidden)
	}

	authRows.Close()
}

func login(w http.ResponseWriter, r *http.Request) {
	var id int
	var hashPassword string
	var auth Authentication

	w.Header().Set("Access-Control-Allow-Origin", "*")
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(&auth)
	defer r.Body.Close()

	if auth.Username == "" || auth.Password == "" {
		return
	}

	rows, err := db.Query(RemoveOldSessions)
	logIfErr(err)
	rows.Close()

	rows, err = db.Query(SelectUserAuth, auth.Username)
	logIfErr(err)

	rows.Next()
	err = rows.Scan(&id, &hashPassword)
	logIfErr(err)
	rows.Close()

	err = bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(auth.Password))

	if err == nil {
		sessionId := createHash()
		rows, err = db.Query(AddSession, id, sessionId)
		logIfErr(err)
		rows.Close()

		exp := time.Now().Add(time.Hour)
		cookie := http.Cookie{Name: "SESSIONID", Value: sessionId, Path: "/", Expires: exp, HttpOnly: true}
		http.SetCookie(w, &cookie)
	} else {
		http.Error(w, "Incorrect username or password", 401)
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	cookie, err := r.Cookie("SESSIONID")
	sessionId := cookie.Value

	rows, err := db.Query(RemoveSession, sessionId)
	logIfErr(err)
	rows.Close()
}

func getThumbnail(artist string, vidId string) {
	var buffer bytes.Buffer

	width := 640
	height := 360
	videoPath := fmt.Sprintf("./data/videos/%s/%s.mp4", artist, vidId)
	thumbnailPath := fmt.Sprintf("./data/thumbnails/%s/%s.jpeg", artist, vidId)
	f, err := os.Create(thumbnailPath)
	logIfErr(err)
	f.Close()

	cmd := exec.Command("ffmpeg", "-i", videoPath, "-vframes", "1", "-s", fmt.Sprintf("%dx%d", width, height), "-f", "singlejpeg", "-")
	cmd.Stdout = &buffer
	err = cmd.Run()
	logIfErr(err)

	thumbnail, err := os.OpenFile(thumbnailPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	logIfErr(err)

	_, err = thumbnail.Write(buffer.Bytes())
	logIfErr(err)
	thumbnail.Close()
}

func addVideo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	cookie, _ := r.Cookie("SESSIONID")

	if !authenticate(cookie) {
		http.Error(w, "Authentication failed", http.StatusForbidden)
		return
	}

	artist := getUserId(cookie.Value)

	file, _, err := r.FormFile("file")
	logServerErr(w, err)

	name := r.FormValue("name")
	desc := r.FormValue("desc")
	genre := r.FormValue("genre")
	data, err := ioutil.ReadAll(file)
	logServerErr(w, err)

	rows, err := db.Query(AddVideo, artist, name, desc, genre)
	logServerErr(w, err)
	rows.Close()

	if err == nil {
		var videoId string

		vidRows, err := db.Query(SelectNewVideoId, artist, name)
		logServerErr(w, err)

		vidRows.Next()
		err = vidRows.Scan(&videoId)
		logIfErr(err)
		vidRows.Close()

		filePath := fmt.Sprintf("./data/videos/%s/%s.mp4", artist, videoId)

		f, err := os.Create(filePath)
		logServerErr(w, err)

		_, err = f.Write(data)
		logServerErr(w, err)
		f.Close()

		getThumbnail(artist, videoId)
	} else {
		http.Error(w, "Server error", http.StatusInternalServerError)
	}
}

func addAvatar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	cookie, _ := r.Cookie("SESSIONID")

	if !authenticate(cookie) {
		http.Error(w, "Authentication failed", http.StatusForbidden)
		return
	}

	artist := getUserId(cookie.Value)

	file, _, err := r.FormFile("file")
	logServerErr(w, err)

	data, err := ioutil.ReadAll(file)
	logServerErr(w, err)

	filePath := fmt.Sprintf("./data/avatars/%s/avatar.jpeg", artist)

	f, err := os.Create(filePath)
	logServerErr(w, err)

	_, err = f.Write(data)
	logServerErr(w, err)
	f.Close()
}

func searchByZipCode(w http.ResponseWriter, r *http.Request) {
	var a BasicArtist
	var artistList []BasicArtist

	w.Header().Set("Access-Control-Allow-Origin", "*")
	cookie, _ := r.Cookie("SESSIONID")

	if !authenticate(cookie) {
		http.Error(w, "Authentication failed", http.StatusForbidden)
		return
	}

	artistId := getUserId(cookie.Value)
	rows, err := db.Query(SelectArtistByZip, artistId)
	checkErr(err)

	for rows.Next() {
		err = rows.Scan(&a.Id, &a.Name, &a.Username)
		logIfErr(err)

		artistList = append(artistList, a)
	}
	rows.Close()

	err = json.NewEncoder(w).Encode(artistList)
	logServerErr(w, err)
}

func searchByCity(w http.ResponseWriter, r *http.Request) {
	var a BasicArtist
	var artistList []BasicArtist

	w.Header().Set("Access-Control-Allow-Origin", "*")
	cookie, _ := r.Cookie("SESSIONID")

	if !authenticate(cookie) {
		http.Error(w, "Authentication failed", http.StatusForbidden)
		return
	}

	artistId := getUserId(cookie.Value)
	rows, err := db.Query(SelectArtistByCity, artistId)
	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&a.Id, &a.Name, &a.Username)
		logIfErr(err)

		artistList = append(artistList, a)
	}

	err = json.NewEncoder(w).Encode(artistList)
	logServerErr(w, err)
}

func getMessages(w http.ResponseWriter, r *http.Request) {
	var messages []Message

	cookie, _ := r.Cookie("SESSIONID")

	if !authenticate(cookie) {
		http.Error(w, "Authentication failed", http.StatusForbidden)
		return
	}
	artistId := getUserId(cookie.Value)
	receiver := r.URL.Query().Get("artist")

	rows, err := db.Query(SelectChatThread, artistId, receiver)
	logIfErr(err)

	for rows.Next() {
		var id string
		var m Message

		err = rows.Scan(&id, &m.Artist, &m.Receiver, &m.Message, &m.Time)
		logIfErr(err)
		updateRows, err := db.Query(UpdateSent, id)
		logIfErr(err)
		updateRows.Close()
		messages = append(messages, m)
	}
	err = rows.Err()
	rows.Close()

	err = json.NewEncoder(w).Encode(messages)
	logServerErr(w, err)
}

func getRecentMessages(w http.ResponseWriter, r *http.Request) {
	var messages []Message
	var id string

	cookie, _ := r.Cookie("SESSIONID")

	if !authenticate(cookie) {
		http.Error(w, "Authentication failed", http.StatusForbidden)
		return
	}
	artistId := getUserId(cookie.Value)
	receiver := r.URL.Query().Get("artist")
	rows, err := db.Query(SelectUnreadM, artistId, receiver)
	logIfErr(err)

	for rows.Next() {
		var m Message
		err = rows.Scan(&id, &m.Artist, &m.Receiver, &m.Message, &m.Time)
		logIfErr(err)
		updateRows, err := db.Query(UpdateSent, id)
		logIfErr(err)
		updateRows.Close()
		messages = append(messages, m)
	}
	rows.Close()

	err = json.NewEncoder(w).Encode(messages)
	logServerErr(w, err)
}

func postMessages(w http.ResponseWriter, r *http.Request) {
	var message PostMessage
	cookie, _ := r.Cookie("SESSIONID")

	if !authenticate(cookie) {
		http.Error(w, "Authentication failed", http.StatusForbidden)
		return
	}

	artistId := getUserId(cookie.Value)
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(&message)

	rows, err := db.Query(AddMessage, artistId, message.Artist, message.Message)
	logIfErr(err)
	rows.Close()
}

func getSharedVideos(w http.ResponseWriter, r *http.Request) {
	var videos VideoList

	w.Header().Set("Access-Control-Allow-Origin", "*")
	cookie, err := r.Cookie("SESSIONID")

	if !authenticate(cookie) {
		http.Error(w, "Authentication failed", http.StatusForbidden)
		return
	}

	rows, err := db.Query(SelectSharedVideos)
	logIfErr(err)

	for rows.Next() {
		var v Video
		var artistId string

		err = rows.Scan(&v.Id, &v.Title, &v.Desc, &artistId, &v.Time, &v.Views, &v.Likes, &v.Genre)
		logIfErr(err)

		var a BasicArtist
		artistRow, aErr := db.Query(SelectBasicArtistData, artistId)
		logIfErr(aErr)

		artistRow.Next()
		err = artistRow.Scan(&a.Id, &a.Username, &a.Name)
		logIfErr(err)
		artistRow.Close()

		v.Artist = a

		videos.VideoCards = append(videos.VideoCards, v)
	}
	rows.Close()

	err = json.NewEncoder(w).Encode(videos)
	logServerErr(w, err)
}

func shareVideo(w http.ResponseWriter, r *http.Request) {
	var v Video
	var artistId string

	w.Header().Set("Access-Control-Allow-Origin", "*")
	cookie, _ := r.Cookie("SESSIONID")

	if !authenticate(cookie) {
		http.Error(w, "Authentication failed", http.StatusForbidden)
		return
	}

	artist := getUserId(cookie.Value)
	fileId := r.FormValue("video")

	rows, err := db.Query(SelectVideoById, fileId)
	rows.Next()
	err = rows.Scan(&v.Id, &v.Title, &v.Desc, &artistId, &v.Time, &v.Views, &v.Likes, &v.Genre)
	rows.Close()

	rows, err = db.Query(ShareVideo, v.Id, v.Title, v.Desc, artist, v.Time, v.Views, v.Likes, v.Genre)
	logIfErr(err)

	oldFile := fmt.Sprintf("./data/videos/%s/%s.mp4", artistId, v.Id)
	newFile := fmt.Sprintf("./data/videos/%s/%s.mp4", artist, v.Id)
	logServerErr(w, err)

	newF, err := os.Create(newFile)
	oldF, err := os.Open(oldFile)
	logServerErr(w, err)

	_, err = io.Copy(oldF, newF)
	logServerErr(w, err)
	newF.Close()
	oldF.Close()

	getThumbnail(artist, v.Id)
}
