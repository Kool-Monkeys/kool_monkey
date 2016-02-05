package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	DB *sql.DB
)

type Configuration struct {
	DbConn DbConnection
}

type DbConnection struct {
	Host string
	Port int
	Name string
	User string
}

type Result struct {
	AgentId     int64  `json:"agentId"`
	TestId      int64  `json:"testId"`
	TestRuntime int64  `json:"testRuntime"`
	TestResults string `json:"testResults"`
	Url         string `json:"url"`
}

type TestCount struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

type AliveResult struct {
	AgentId interface{}              `json:"agentId"`
	Status  string                   `json:"status"`
	Message string                   `json:"message"`
	Jobs    []map[string]interface{} `json:"jobs"`
}

type queryResult struct {
	TestId interface{}              `json:"testId"`
	Result []map[string]interface{} `json:"results"`
}

type TestSite struct {
	TestId    int    `json:"test_id"`
	TargetUrl string `json:"target_url"`
	Frequency int    `json:"frequency"`
}

type Agent struct {
	AgentId   int       `json:"agent_id"`
	Ip        string    `json:"ip"`
	LastAlive time.Time `json:"last_alive"`
}

func connectToDb(db DbConnection) error {
	var err error
	connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s sslmode=disable", db.Host, db.Port, db.Name, db.User)
	DB, err = sql.Open("postgres", connStr)
	return err
}

func result(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	message := make(map[string]string)

	dec := json.NewDecoder(r.Body)
	var resultData Result
	err := dec.Decode(&resultData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		message["message"] = "Invalid request"
		enc.Encode(&message)
		return
	}

	_, err = DB.Exec(
		"INSERT INTO result (agentId, testId, testRuntime, testResults) VALUES ($1, $2, $3, $4)",
		resultData.AgentId,
		resultData.TestId,
		resultData.TestRuntime,
		resultData.TestResults,
	)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		message["message"] = "Couldn't save result"
		enc.Encode(&message)
		return
	}

	w.WriteHeader(http.StatusOK)
	message["message"] = "Correctly saved"
	enc.Encode(&message)
}

func query(w http.ResponseWriter, r *http.Request) {
	fmtDate := make([]time.Time, 2)
	vars := mux.Vars(r)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)

	var response queryResult
	response.TestId = vars["testId"]
	const longForm = "Mon Jan 2 15:04:05 -0700 MST 2006"

	dateFrom := r.FormValue("dateFrom")
	if dateFrom != "" {
		fmtDate[0], _ = time.Parse(longForm, dateFrom)
	}

	dateTo := r.FormValue("dateTo")
	if dateTo != "" {
		fmtDate[1], _ = time.Parse(longForm, dateTo)
	}

	// Checking the date format
	if dateFrom != "" && fmtDate[0].Unix() == -62135596800 {
		enc.Encode(&response)
		fmt.Println("dateFrom: Format error")
		return
	} else if dateTo != "" && fmtDate[1].Unix() == -62135596800 {
		enc.Encode(&response)
		fmt.Println("dateTo: Format error")
		return
	}

	// Checking that dateFrom <= dateTo
	extraQuery := ""
	if dateFrom != "" && dateTo != "" && fmtDate[0].Unix() > fmtDate[1].Unix() {
		enc.Encode(&response)
		fmt.Println("dateFom is more recent than dateTo")
		return
	} else if dateFrom != "" && dateTo != "" {
		const timestamp = "2014-01-22 12:22:30"
		extraQuery = fmt.Sprintf(" AND timestamp BETWEEN '%s' AND '%s'", fmtDate[0].Format(timestamp), fmtDate[1].Format(timestamp))
	}

	rows, errQuery := DB.Query("SELECT result.id, result.agentId, test.targetUrl, result.testRuntime, result.timestamp FROM result INNER JOIN test ON test.id = result.testId WHERE test.id = $1" + extraQuery, vars["testId"])
	if errQuery == nil {
		var id int
		var agentId int
		var url string
		var responseTime int
		var timestamp time.Time

		for i := 0; rows.Next(); i++ {
			result := make(map[string]interface{})
			rows.Scan(&id, &agentId, &url, &responseTime, &timestamp)

			result["id"] = id
			result["agentId"] = agentId
			result["url"] = url
			result["responseTime"] = responseTime
			result["timestamp"] = timestamp

			// If Result is full it must grow.
			if i == cap(response.Result) {
				newSlice := make([]map[string]interface{}, len(response.Result), 2*len(response.Result)+1)
				copy(newSlice, response.Result)
				response.Result = newSlice
			}

			response.Result = append(response.Result, result)
		}
		rows.Close()
	} else {
		fmt.Print(errQuery)
	}

	enc.Encode(&response)
}

func alive(w http.ResponseWriter, r *http.Request) {
	var agentOk bool
	var err error

	// Read the JSON from the request.
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	var dat map[string]interface{}
	if err = json.Unmarshal(buf.Bytes(), &dat); err != nil {
		panic(err)
	}

	// Insert/update in the database the agent information.
	ip := strings.Split(r.RemoteAddr, ":")[0]
	id, ok := dat["agentId"]
	if ok {
		_, err = DB.Exec("UPDATE agent SET ip = $1, lastAlive = now() WHERE id = $2", ip, dat["agentId"])
		agentOk = (err == nil)
	} else {
		err = DB.QueryRow("INSERT INTO agent (ip, lastAlive) VALUES ($1, now()) RETURNING id", ip).Scan(&id)
		agentOk = (err == nil)
	}

	// Prepare and send the response to the agent
	var response AliveResult
	if agentOk {
		response.AgentId = id
		response.Status = "OK"
		w.WriteHeader(http.StatusOK)

		rows, errQuery := DB.Query("SELECT test.id, test.targetURL, test.frequency FROM test INNER JOIN testAgent ON test.id = testAgent.idTest WHERE testAgent.idAgent = $1", id)
		if errQuery == nil {
			var testId int
			var targetUrl string
			var frecuency int
			for i := 0; rows.Next(); i++ {
				job := make(map[string]interface{})
				rows.Scan(&testId, &targetUrl, &frecuency)

				// If Jobs is full it must grow.
				if i == cap(response.Jobs) {
					newSlice := make([]map[string]interface{}, len(response.Jobs), 2*len(response.Jobs)+1)
					copy(newSlice, response.Jobs)
					response.Jobs = newSlice
				}

				job["testId"] = testId
				job["targetURL"] = targetUrl
				job["frequency"] = frecuency

				response.Jobs = append(response.Jobs, job)
			}
			rows.Close()
		} else {
			fmt.Print(errQuery)
		}
	} else {
		fmt.Print(err)
		response.AgentId = -1
		response.Status = "KO"
		response.Message = "Couldn't update the agent"
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(&response)
}

func addSite(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	response := make(map[string]interface{})

	var okUrl, okFreq bool
	var targetUrl string
	var frequency float64
	var testId int

	dec := json.NewDecoder(r.Body)
	siteTest := make(map[string]interface{})
	err := dec.Decode(&siteTest)
	if err == nil {
		targetUrl, okUrl = siteTest["targetUrl"].(string)
		frequency, okFreq = siteTest["frequency"].(float64)
	}
	if err != nil || !okUrl || !okFreq {
		w.WriteHeader(http.StatusBadRequest)
		response["status"] = "KO"
		response["message"] = "Invalid JSON"
		enc.Encode(&response)
		return
	}

	err = DB.QueryRow(
		"INSERT INTO test (targetUrl, frequency) VALUES ($1, $2) RETURNING id",
		targetUrl,
		int(frequency),
	).Scan(&testId)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		response["status"] = "KO"
		response["message"] = "Couldn't save test"
		enc.Encode(&response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response["status"] = "OK"
	response["message"] = "Correctly saved"
	response["testId"] = testId
	enc.Encode(&response)
}

func getSites(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	response := make(map[string]interface{})

	testId := 0
	testIdStr := r.FormValue("test_id")
	if testIdStr != "" {
		var err error
		testId, err = strconv.Atoi(testIdStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response["status"] = "KO"
			response["message"] = "Invalid test ID"
			enc.Encode(&response)
			return
		}
	}

	rows, err := DB.Query("SELECT id, targetUrl, frequency FROM test WHERE id = $1 OR $1 = 0", testId)
	defer rows.Close()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response["status"] = "KO"
		response["message"] = "Couldn't get tests"
		enc.Encode(&response)
		return
	}

	tests := make([]TestSite, 0)
	for rows.Next() {
		var testSite TestSite
		rows.Scan(&testSite.TestId, &testSite.TargetUrl, &testSite.Frequency)
		tests = append(tests, testSite)
	}

	w.WriteHeader(http.StatusOK)
	response["status"] = "OK"
	response["test_sites"] = tests
	enc.Encode(&response)
}

func getAgents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	response := make(map[string]interface{})

	agentId := 0
	agentIdStr := r.FormValue("agent_id")
	if agentIdStr != "" {
		var err error
		agentId, err = strconv.Atoi(agentIdStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response["status"] = "KO"
			response["message"] = "Invalid agent ID"
			enc.Encode(&response)
			return
		}
	}

	rows, err := DB.Query("SELECT id, ip, lastAlive FROM agent WHERE id = $1 OR $1 = 0", agentId)
	defer rows.Close()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response["status"] = "KO"
		response["message"] = "Couldn't get agents"
		enc.Encode(&response)
		return
	}

	agents := make([]Agent, 0)
	for rows.Next() {
		var a Agent
		rows.Scan(&a.AgentId, &a.Ip, &a.LastAlive)
		agents = append(agents, a)
	}

	w.WriteHeader(http.StatusOK)
	response["status"] = "OK"
	response["agents"] = agents
	enc.Encode(&response)
}

func getTestsPerHour(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	response := make(map[string]interface{})

	date := r.FormValue("date")

	rows, err := DB.Query("select date_part('hour', timestamp) as hour, count(*) as count from result where date_trunc('day', timestamp) = $1::timestamp group by date_part('hour', timestamp) order by date_part('hour', timestamp)", date)
	defer rows.Close()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response["status"] = "KO"
		response["message"] = "Couldn't get test count"
		enc.Encode(&response)
		return
	}

	tests := make([]TestCount, 0)
	for rows.Next() {
		var tc TestCount
		rows.Scan(&tc.Hour, &tc.Count)
		tests = append(tests, tc)
	}

	w.WriteHeader(http.StatusOK)
	response["status"] = "OK"
	response["tests"] = tests
	enc.Encode(&response)
}

func main() {
	koolDir, err := filepath.Abs(filepath.Dir(os.Args[0]) + "/../")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Starting static dashboard server at port 3002")
	go func() {
		panic(http.ListenAndServe(":3002", http.FileServer(http.Dir(koolDir+"/dashboard"))))
	}()

	fmt.Println("Starting static www server at port 3001")
	go func() {
		panic(http.ListenAndServe(":3001", http.FileServer(http.Dir(koolDir+"/www"))))
	}()

	fmt.Println("Starting api server at port 3000")

	//Read config
	cmd_cfg := flag.String("conf", koolDir+"/conf/kool-server.conf", "Config file")
	flag.Parse()
	file, err := os.Open(*cmd_cfg)
	if err != nil {
		fmt.Printf("Config - File Error: %s\n", err)
		os.Exit(1)
	}

	decoder := json.NewDecoder(file)
	conf := Configuration{}

	if err := decoder.Decode(&conf); err != nil {
		fmt.Printf("Config - Decoding Error: %s\n", err)
		os.Exit(1)
	}

	//Connect to DB
	err = connectToDb(conf.DbConn)
	if err != nil {
		fmt.Println("Couldn't connect to DB!")
		os.Exit(1)
	}

	/* Initialize handlers */
	router := mux.NewRouter()
	router.HandleFunc("/result", result).Methods("POST")
	router.HandleFunc("/alive", alive).Methods("POST")
	router.HandleFunc("/sites", addSite).Methods("POST")
	router.HandleFunc("/sites", getSites).Methods("GET")
	router.HandleFunc("/agents", getAgents).Methods("GET")
	router.HandleFunc("/query/{testId}", query).Methods("GET")
	router.HandleFunc("/tests", getTestsPerHour).Methods("GET")

	n := negroni.Classic()
	n.UseHandler(router)
	n.Run(":3000")
}
