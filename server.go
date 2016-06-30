package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

//sensors configuration
const (
	// ON sensor activated
	ON = true
	// OFF sensor deactivated
	OFF = false
)

//sensors state after test
const (
	//DISSABLED
	DISSABLED = 0
	//RUNNING
	READY = 1
	//BROKEN
	BROKEN = 2
)

// StaticURL URL of the static content
const StaticURL string = "/static/"

// StaticRoot path of the static content
const StaticRoot string = "static/"

// DataFilePath path of the data files on StaticRoot
const DataFilePath string = "data/"

// DataExtension extension of the data files
const DataExtension = ".csv"

//level of attention of the messages
const (
	HIDE    = 0
	INFO    = 1
	SUCCESS = 2
	WARNING = 3
	DANGER  = 4
)

//state of the system
const (
	INIT       = 0
	CONFIGURED = 1
	RUNNING    = 2
	STOPPED    = 3
)

//Context data about the configuration of the system and the web page
type Context struct {
	//web page related
	Title  string
	Static string
	//configuration of the system
	ConfigurationName string
	// DataFilePath
	DataFile *os.File
	//state of the processed
	State int //INIT, CONFIGURED, RUNNING, STOPPED
	//message
	Message    string
	AlertLevel int // HIDE, INFO, SUCCESS, WARNING, DANGER

	//settings of the sensors: ON or OFF
	SetTrackerA bool
	SetTrackerB bool
	SetTrackerC bool
	SetTrackerD bool
	SetTrackerM bool
	SetDistance bool
	SetAccGyro  bool
	// state of sensor after test calling
	StateOfTrackerA int
	StateOfTrackerB int
	StateOfTrackerC int
	StateOfTrackerD int
	StateOfTrackerM int
	StateOfDistance int
	StateOfAccGyro  int

	//datafiles in the data directory
	DataFiles []string
}

//All the context of the execution with system and web data
var context Context

//RemoveContents erase the contents of a directory
//intended to remove data files en data directory
func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

//Home of the website
func Home(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", context)

	context.Title = "Welcome!"
	render(w, "index", context)
}

//ThePlatform describes the system
func ThePlatform(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", context)

	context.Message = "Description of the Platform"
	context.AlertLevel = INFO
	context.Title = "The Platform"
	render(w, "thePlatform", context)
}

//Init set the platform in a initial state
func Init(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", context)

	switch context.State {
	case INIT, CONFIGURED, STOPPED:
		// correct states
		if req.Method == "GET" {
			context.Message = "Warning! You are erasing the configuration, the datafiles and restoring the platform to it's initial state."
			context.AlertLevel = DANGER
			context.Title = "Initialization"
			render(w, "init", context)
		} else { // POST
			fmt.Println("POST")
			req.ParseForm()
			fmt.Println(req.Form)
			if req.Form.Get("initializate") == "YES" {
				//if YES, init the platform
				context.State = INIT
				context.ConfigurationName = ""
				//erase datafiles
				dataDirectory := filepath.Join(StaticRoot, DataFilePath)
				fmt.Println("DELETING ", dataDirectory)
				err := RemoveContents(dataDirectory)
				if err != nil {
					fmt.Println(err)
				}

				//message of initial state
				context.Message = "The system is now in the initial state. Now you must define a new configuration berofe run an experiment."
				context.AlertLevel = SUCCESS
			} else {
				//message of initial state
				context.Message = "The system initialization is canceled. The current configuration is active."
				context.AlertLevel = WARNING
			}
			//initiated or not, shows the experiment page
			context.Title = "Experiment"
			render(w, "experiment", context)
		}
	case RUNNING:
		// worng state
		context.Message = "System is running! It MUST be stopped before erase the configuration and set the initial state."
		context.AlertLevel = DANGER
		context.Title = "Run"
		render(w, "run", context)
	}

}

//Experiment allows to access to the experiments
func Experiment(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", context)

	switch context.State {
	case INIT, CONFIGURED, STOPPED:
		//correct cases, shows the experiment page to config,test and run it
		context.Message = "Let's make some experiments"
		context.AlertLevel = INFO
		context.Title = "Experiment"
		render(w, "experiment", context)
	case RUNNING:
		//wrong case, it must be STOPPED before
		context.Message = "System is running! It MUST be stopped before a new configuration done."
		context.AlertLevel = DANGER
		context.Title = "Run"
		render(w, "run", context)
	}
}

//Config allows to configure the sensors
func Config(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", context)

	switch context.State {
	case INIT, CONFIGURED, STOPPED:
		//correct states, do the config process
		if req.Method == "GET" {
			context.Message = "Activate/Deactivate the sensors."
			context.AlertLevel = INFO
			context.Title = "Configuration of Sensor Platform"
			render(w, "config", context)
		} else { // POST
			fmt.Println("POST")
			req.ParseForm()
			// logic part of login
			//validation phase will be here
			//if valid, put the form data into the context struct
			context.ConfigurationName = req.Form.Get("ConfigurationName")
			if req.Form.Get("SetTrackerA") == "on" {
				context.SetTrackerA = ON
			} else {
				context.SetTrackerA = OFF
			}
			if req.Form.Get("SetTrackerB") == "on" {
				context.SetTrackerB = ON
			} else {
				context.SetTrackerB = OFF
			}
			if req.Form.Get("SetTrackerC") == "on" {
				context.SetTrackerC = ON
			} else {
				context.SetTrackerC = OFF
			}
			if req.Form.Get("SetTrackerD") == "on" {
				context.SetTrackerD = ON
			} else {
				context.SetTrackerD = OFF
			}
			if req.Form.Get("SetTrackerM") == "on" {
				context.SetTrackerM = ON
			} else {
				context.SetTrackerM = OFF
			}
			if req.Form.Get("SetDistance") == "on" {
				context.SetDistance = ON
			} else {
				context.SetDistance = OFF
			}
			if req.Form.Get("SetAccGyro") == "on" {
				context.SetAccGyro = ON
			} else {
				context.SetAccGyro = OFF
			}
			//prepare the context
			context.Message = "Configuration done! Now the system can be tested or runned the experiment"
			context.AlertLevel = SUCCESS
			context.State = CONFIGURED
			context.Title = "Experiment"
			//log
			fmt.Println(req.Form)
			fmt.Println("Contex:", context)
			//once processed the form, reditect to the index page

			//render(w, "experiment", context)
			http.Redirect(w, req, "/experiment/", http.StatusFound)
		}
	case RUNNING:
		// only put a message, but don't touch the running process
		context.Message = "System is running! It MUST be stopped before a new configuration done."
		context.AlertLevel = DANGER
		context.Title = "Run"
		render(w, "run", context)
	}
}

//Test allows to test the sensors
func Test(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", context)

	switch context.State {
	case INIT:
		//The system must be configured before
		context.Message = "The system must be configured before you could test it!"
		context.AlertLevel = WARNING
		context.Title = "Configure"
		render(w, "configure", context)
	case RUNNING:
		//wrong state, the system must be stopped before
		context.Message = "Warning! You must stop the system before test the system."
		context.AlertLevel = DANGER
		context.Title = "Run"
		render(w, "run", context)
	case CONFIGURED, STOPPED:
		//correct state, let's test the system, and then to experiment page

		//check state of the sensors and put it on stateOfSensors
		//put here the test code
		//put here the test code
		//put here the test code
		//put here the test code

		// this test is a naive one, only for demonstration purpose
		if context.SetTrackerA {
			context.StateOfTrackerA = READY
		} else {
			context.StateOfTrackerA = DISSABLED
		}
		if context.SetTrackerB {
			context.StateOfTrackerB = READY
		} else {
			context.StateOfTrackerB = DISSABLED
		}
		if context.SetTrackerC {
			context.StateOfTrackerC = READY
		} else {
			context.StateOfTrackerC = DISSABLED
		}
		if context.SetTrackerD {
			context.StateOfTrackerD = READY
		} else {
			context.StateOfTrackerD = DISSABLED
		}
		if context.SetTrackerM {
			context.StateOfTrackerM = READY
		} else {
			context.StateOfTrackerM = DISSABLED
		}
		if context.SetDistance {
			context.StateOfDistance = READY
		} else {
			context.StateOfDistance = DISSABLED
		}
		if context.SetAccGyro {
			context.StateOfAccGyro = READY
		} else {
			context.StateOfAccGyro = DISSABLED
		}
		// test done, shows the result

		context.Title = "Test the Sensor Platform"
		context.Message = "System configured and Tested. Ready to run."
		context.AlertLevel = SUCCESS
		fmt.Println(">>>", context)
		render(w, "test", context)
	}
}

//Run allows to run the experiments
func Run(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", context)

	switch context.State {
	case INIT:
		//wrong state, show experiment page
		context.Message = "Warning! You must configure the system before run the experiment."
		context.AlertLevel = DANGER
		context.Title = "experiment"
		//http.Redirect(w, req, "/experiment/", http.StatusFound)
		render(w, "experiment", context)
	case CONFIGURED, STOPPED:
		//correct states, do the running process

		dataFileName := filepath.Join(StaticRoot, DataFilePath, context.ConfigurationName+DataExtension)
		//detect if file exists
		_, err := os.Stat(dataFileName)
		//create datafile is not exists
		if os.IsNotExist(err) {
			//create file to write
			fmt.Println("Creating ", dataFileName)
			context.DataFile, err = os.Create(dataFileName)
			if err != nil {
				fmt.Println(err.Error())
			}
		} else {
			//open fle to append
			fmt.Println("Openning ", dataFileName)
			context.DataFile, err = os.OpenFile(dataFileName, os.O_RDWR|os.O_APPEND, 0644)
			if err != nil {
				fmt.Println(err.Error())
			}
		}

		// running process instruction here!
		// running process instruction here!
		// running process instruction here!

		//dump the data gathered in DataFile
		_, err = context.DataFile.WriteString("123, 123, 123, 123\n")
		err = context.DataFile.Sync()
		if err != nil {
			fmt.Println(err.Error())
		}
		//defer close the file to STOP

		context.Message = "System running gathering data from sensors."
		context.AlertLevel = SUCCESS
		context.Title = "Run"
		context.State = RUNNING
		render(w, "run", context)
	case RUNNING:
		// we already are in this State
		// only put a message, but don't touch the running process
		context.Message = "System is ALREADY running!"
		context.AlertLevel = WARNING
		context.Title = "Run"
		render(w, "run", context)
	}
}

//Stop allows to stop the experiments
func Stop(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", context)

	switch context.State {
	case INIT, CONFIGURED:
		context.Message = "Warning! You must configure the system and run the experiment before stop it."
		context.AlertLevel = DANGER
		context.Title = "Experiment"
		render(w, "experiment", context)
	case RUNNING:
		//correct state, do the stop process
		// stop process instruction here!
		// stop process instruction here!
		// stop process instruction here!

		//close the file
		err := context.DataFile.Sync()
		if err != nil {
			fmt.Println(err.Error())
		}
		context.DataFile.Close()

		context.Title = "Stop"
		context.State = STOPPED
		context.Message = "System stopped. Now you can donwload the data to your permanent storage"
		context.AlertLevel = SUCCESS
		render(w, "stop", context)

	case STOPPED:
		// we already are in this State
		// only put a message, but don't touch the process
		context.Message = "System is ALREADY stooped!"
		context.AlertLevel = WARNING
		context.Title = "Stop"
		render(w, "experiment", context)
	}
}

//Collect the data gathered in the experiments
func Collect(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", context)

	switch context.State {
	case INIT, CONFIGURED, STOPPED:
		//read the data directory and offers the files to be downloaded
		context.DataFiles, _ = filepath.Glob(filepath.Join(StaticRoot, DataFilePath, "*"+DataExtension))
		//fmt.Println(">>>> " + filepath.Join(StaticRoot, DataFilePath, "*"+DataExtension))
		//let only the file name, eliminate the path
		for i, f := range context.DataFiles {
			context.DataFiles[i] = path.Base(f)
		}

		fmt.Println(context.DataFiles)

		context.Title = "Collect Data"
		if len(context.DataFiles) == 0 {
			context.Message = "Sorry! There are not files to donwload stored in the system."
			context.AlertLevel = WARNING
		} else {
			context.Message = "You can download the data stored in the system."
			context.AlertLevel = INFO
		}
		render(w, "collect", context)
	case RUNNING:
		context.Message = "You can't download data is while the system is running. You must stop the system before."
		context.AlertLevel = WARNING
		context.Title = "Run"
		render(w, "run", context)
	}

}

//About shows the page with info
func About(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", context)

	context.Title = "About"
	render(w, "about", context)
}

//Help shows information about the tool
func Help(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", context)

	context.Title = "Help"
	render(w, "help", context)
}

// render
func render(w http.ResponseWriter, tmpl string, context Context) {
	fmt.Println("[render]>>>", context)
	context.Static = StaticURL
	//list of templates, put here all the templates needed
	tmplList := []string{"templates/base.html",
		fmt.Sprintf("templates/message.html"),
		fmt.Sprintf("templates/%s.html", tmpl)}
	t, err := template.ParseFiles(tmplList...)
	if err != nil {
		log.Print("template parsing error: ", err)
	}
	err = t.Execute(w, context)
	if err != nil {
		log.Print("template executing error: ", err)
	}
}

//StaticHandler allows to server the statics references
func StaticHandler(w http.ResponseWriter, req *http.Request) {
	staticFile := req.URL.Path[len(StaticURL):]
	if len(staticFile) != 0 {
		f, err := http.Dir(StaticRoot).Open(staticFile)
		if err == nil {
			content := io.ReadSeeker(f)
			http.ServeContent(w, req, staticFile, time.Now(), content)
			return
		}
	}
	http.NotFound(w, req)
}

func main() {
	//set the initial state
	context.State = INIT

	http.HandleFunc("/", Home)
	http.HandleFunc("/thePlatform/", ThePlatform)
	http.HandleFunc("/experiment/", Experiment)
	http.HandleFunc("/init/", Init)
	http.HandleFunc("/config/", Config)
	http.HandleFunc("/test/", Test)
	http.HandleFunc("/run/", Run)
	http.HandleFunc("/stop/", Stop)
	http.HandleFunc("/collect/", Collect)
	http.HandleFunc("/about/", About)
	http.HandleFunc("/help/", Help)
	http.HandleFunc(StaticURL, StaticHandler)
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
