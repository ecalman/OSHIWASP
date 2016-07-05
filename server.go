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

	"bufio"
	"bytes"
	"encoding/binary"
	"runtime"

	"github.com/mrmorphic/hwio"
	"github.com/tarm/serial"
)

//sensors configuration

const (

	// CommDevName arduino serial comm related
	CommDevName = "/dev/rfcomm1" //name of the BT device
	//Bauds Bauds
	Bauds = 9600 // bauds of the BT serial channel

	//StatusLedPin pin which shows the status
	StatusLedPin = "gpio7" // green
	//ActionLedPin pin which shows the action of the system
	ActionLedPin = "gpio8" // yellow

	//ButtonAPin pin
	ButtonAPin = "gpio24" // start
	//ButtonBPin pin
	ButtonBPin = "gpio23" // stop

	//TrackerAPin pin
	TrackerAPin = "gpio22"
	//TrackerBPin pin
	TrackerBPin = "gpio18"
	//TrackerCPin pin
	TrackerCPin = "gpio17"
	//TrackerDPin pin
	TrackerDPin = "gpio4"
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
	//string received by the form configuration
	SensorStateOn  = "on"
	SensorStateOff = "off"
)

//// web related
// tmplPath = "tmpl/" // path of the template files .html in the local file system
// dataPath = "data/" // path of the data files in the local file system
// dataFileExtension = ".csv" //  data file extension in the local file system

// StaticURL URL of the static content
const StaticURL string = "/static/"

// StaticRoot path of the static content
const StaticRoot string = "static/"

// DataFilePath path of the data files on StaticRoot
const DataFilePath string = "data/"

// DataFileExtension extension of the data files
const DataFileExtension string = ".csv"

//level of attention of the messages
const (
	HIDE    = 0
	INFO    = 1
	SUCCESS = 2
	WARNING = 3
	DANGER  = 4
)

//state of the system
//stateNEW = "NEW"
//stateRUNNING = "RUNNING"
//statePAUSED = "PAUSED"
//stateSTOPPED = "STOPPED"
//stateERROR = "ERROR"

//state of system
const (
	INIT       = 0
	CONFIGURED = 1
	RUNNING    = 2
	STOPPED    = 3
)

//title of pages respect of state
const (
	titleWelcome     = "Welcome!"
	titleThePlatform = "The Platform"
	titleInit        = "Initialization"
	titleConfig      = "Configuration of Sensor Platform"
	titleTest        = "Test the Sensor Platform"
	titleExperiment  = "Experiment"
	titleRun         = "Run"
	titleStop        = "Stop"
	titleCollect     = "Collect Data"
	titleAbout       = "About"
	titleHelp        = "Help"
)

//Context data about the configuration of the system and the web page
type Context struct {
	//web page related
	Title  string
	Static string
	//web appearance : message and alert level
	Message    string
	AlertLevel int // HIDE, INFO, SUCCESS, WARNING, DANGER

	//state of the processed
	State int //INIT, CONFIGURED, RUNNING, STOPPED
	//time of acquisition
	Time0 time.Time

	//configuration name of the system
	ConfigurationName string
	// DataFilePath
	DataFile *os.File
	//datafiles in the data directory
	DataFiles []string
	//data file name
	DataFileName string

	//arduino
	SerialPort *serial.Port

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
}

// SensorDataInBytes data for sensors in Arduino in bytes
type SensorDataInBytes struct {
	sincroMicroSecondsInBytes []byte
	sensorMicroSecondsInBytes []byte
	distanceInBytes           []byte
	accXInBytes               []byte
	accYInBytes               []byte
	accZInBytes               []byte
	gyrXInBytes               []byte
	gyrYInBytes               []byte
	gyrZInBytes               []byte
}

// SensorData data for sensors in Arduino in numerical data types
type SensorData struct {
	sincroMicroSeconds uint32
	sensorMicroSeconds uint32
	distance           uint32
	accX               float32
	accY               float32
	accZ               float32
	gyrX               float32
	gyrY               float32
	gyrZ               float32
}

// Oshiwasp definition of configPuration of raspberry sensors, leds and buttons
type Oshiwasp struct {
	statusLed hwio.Pin
	actionLed hwio.Pin
	buttonA   hwio.Pin
	buttonB   hwio.Pin
	trackerA  hwio.Pin
	trackerB  hwio.Pin
	trackerC  hwio.Pin
	trackerD  hwio.Pin
}

// // SensorData data for sensors in Arduino
// type SensorData struct{
//         sincroMicroSeconds uint32
//         sensorMicroSeconds uint32
//         distance uint32
//         accX float32
//         accY float32
//         accZ float32
//         gyrX float32
//         gyrY float32
//         gyrZ float32
// }
// // SensorDataIn Bytes data for sensors in Arduino in bytes
// type SensorDataInBytes struct{
//    sincroMicroSecondsInBytes []byte
// 	sensorMicroSecondsInBytes []byte
// 	distanceInBytes []byte
// 	accXInBytes []byte
// 	accYInBytes []byte
// 	accZInBytes []byte
// 	gyrXInBytes []byte
// 	gyrYInBytes []byte
// 	gyrZInBytes []byte
// }
//
// //Acquisition definition
// type Acquisition struct{
//     outputFileName string
//     outputFile *os.File
//     state string
//     time0 time.Time
//     //arduino
//     serialPort *serial.Port
//     //configuration
//     ConfigurationName string
//     TrackerA          bool
//     TrackerB          bool
//     TrackerC          bool
//     TrackerD          bool
//     TrackerM          bool
//     Distance          bool
//     AccGyro           bool
// }
//
// // StateOfSensors state
// type StateOfSensors struct { // state of the sensors
//         ConfigurationName string
//         TrackerA          string
//         TrackerB          string
//         TrackerC          string
//         TrackerD          string
//         TrackerM          string
//         Distance          string
//         AccGyro           string
// }
//

//comment for free

var (
	c chan int //channel initialitation
	//actionLed hwio.Pin // indicating action in the system

	// templates = template.Must(template.ParseGlob(tmplPath+"*.tmpl"))
	// validPath = regexp.MustCompile("^/(index|new|status|start|pause|resume|stop|download|data)/([a-zA-Z0-9]+)$")

	theSensorData        = new(SensorData)
	theSensorDataInBytes = new(SensorDataInBytes)

	//All the context of the execution with system and web data
	theContext Context //theAcq=new(Acquisition)

	theOshi = new(Oshiwasp)
)

//AAAAAAAAAAAAAA
// Acquisition section
//AAAAAAAAAAAAAA

func (cntxt *Context) connectArduinoSerialBT() {
	// config the comm port for serial via BT
	commPort := &serial.Config{Name: CommDevName, Baud: Bauds}
	// open the serial comm with the arduino via BT
	cntxt.SerialPort, _ = serial.OpenPort(commPort)
	//defer acq.serialPort.Close()
	log.Printf("Open serial device %s", CommDevName)
}

func (cntxt *Context) setArduinoStateON() {
	// activate the readdings in Arduino sending 'ON'
	log.Printf("before write on")
	_, err := cntxt.SerialPort.Write([]byte("n"))
	log.Printf("after write on")
	if err != nil {
		log.Fatal(err)
	}
}

func (cntxt *Context) setArduinoStateOFF() {
	// deactivate the readdings in Artudino sending 'OFF'
	log.Printf("before write off")
	_, err := cntxt.SerialPort.Write([]byte("f"))
	log.Printf("after write off")
	if err != nil {
		log.Printf("error!! after write off")
		log.Fatal(err)
	}
}

func (cntxt *Context) setTime0() {
	cntxt.Time0 = time.Now()
}

func (cntxt *Context) getTime0() time.Time {
	return cntxt.Time0
}

// func (cntxt *Context) getState() int {
// 	return cntxt.State
// }

// func (cntxt *Context) setState(s int) {
// 	cntxt.State = s
// 	log.Printf("State set to %d\n", cntxt.State)
// }

// func (acq *Acquisition) setStateNEW() {
// 	acq.state = stateNEW
// 	acq.setArduinoStateOFF()
//
// 	log.Printf("State: NEW")
// }

// func (acq *Acquisition) setStateRUNNING() {
// 	acq.state = stateRUNNING
// 	acq.setArduinoStateON()
// 	log.Printf("State: RUNNING")
// }

// func (acq *Acquisition) setStatePAUSED() {
// 	acq.state = statePAUSED
// 	acq.setArduinoStateOFF()
// 	log.Printf("State: PAUSED")
// }

// func (acq *Acquisition) setStateSTOPPED() {
// 	acq.state = stateSTOPPED
// 	acq.setArduinoStateOFF()
// 	log.Printf("State: STOPPED")
// }

// func (acq *Acquisition) setStateERROR() {
// 	acq.state = stateERROR
// 	acq.setArduinoStateOFF()
// 	log.Printf("State: ERROR")
// }

// // set the output file name based in the configurationName
// func (acq *Acquisition) setOutputFileName(s string) {
// 	acq.outputFileName = s
// 	log.Printf("Output Filename set to %s\n", acq.outputFileName)
// }

func (cntxt *Context) createOutputFile() {
	var e error
	cntxt.DataFileName = DataFilePath + cntxt.ConfigurationName + DataFileExtension
	cntxt.DataFile, e = os.Create(cntxt.DataFileName)
	if e != nil {
		panic(e)
	}
	statusLine := fmt.Sprintf("### %v Data Acquisition: %s \n\n", time.Now(), cntxt.ConfigurationName)
	cntxt.DataFile.WriteString(statusLine)
	formatLine := fmt.Sprintf("### [Ard], localTime(us), sincroTime(us), sensorTime(us), distance(mm), accX(g), accY(g), accZ(g), gyrX(gr/s), gyrY(gr/s), gyrZ(gr/s) \n\n")
	cntxt.DataFile.WriteString(formatLine)

	log.Printf("Cretated output File %s", cntxt.DataFileName)
}

// func (acq *Acquisition) reopenOutputFile() {
// 	var e error
// 	acq.outputFile, e = os.OpenFile(acq.outputFileName, os.O_WRONLY|os.O_APPEND, 0666)
// 	if e != nil {
// 		panic(e)
// 	}
// 	log.Printf("Reopen output File %s", acq.outputFileName)
// }

// func (acq Acquisition) closeOutputFile() { //close the output file
// 	acq.outputFile.Close()
// 	log.Printf("Closed output File %s", acq.outputFileName)
// }

func (cntxt *Context) initiate() {
	//acq.setOutputFileName(dataPath+dataFileName+dataFileExtension)
	//acq.createOutputFile()
	cntxt.connectArduinoSerialBT()
	log.Printf("Arduino connected!")
	//cntxt.setStateNEW()
}

//OOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO
// Oshiwasp section: Raspberry sensors
//OOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO

func (oshi *Oshiwasp) initiate() {

	var e error
	// Set up 'trakers' as inputs
	oshi.trackerA, e = hwio.GetPinWithMode(TrackerAPin, hwio.INPUT)
	if e != nil {
		panic(e)
	}
	log.Printf("Set pin %s as trackerA\n", TrackerAPin)

	oshi.trackerB, e = hwio.GetPinWithMode(TrackerBPin, hwio.INPUT)
	if e != nil {
		panic(e)
	}
	log.Printf("Set pin %s as trackerB\n", TrackerBPin)

	oshi.trackerC, e = hwio.GetPinWithMode(TrackerCPin, hwio.INPUT)
	if e != nil {
		panic(e)
	}
	log.Printf("Set pin %s as trackerC\n", TrackerCPin)

	oshi.trackerD, e = hwio.GetPinWithMode(TrackerDPin, hwio.INPUT)
	if e != nil {
		panic(e)
	}
	log.Printf("Set pin %s as trackerD\n", TrackerDPin)

	// Set up 'buttons' as inputs
	oshi.buttonA, e = hwio.GetPinWithMode(ButtonAPin, hwio.INPUT)
	if e != nil {
		panic(e)
	}
	log.Printf("Set pin %s as buttonA\n", ButtonAPin)

	oshi.buttonB, e = hwio.GetPinWithMode(ButtonBPin, hwio.INPUT)
	if e != nil {
		panic(e)
	}
	log.Printf("Set pin %s as buttonB\n", ButtonBPin)

	// Set up 'leds' as outputs
	oshi.statusLed, e = hwio.GetPinWithMode(StatusLedPin, hwio.OUTPUT)
	if e != nil {
		panic(e)
	}
	log.Printf("Set pin %s as statusLed\n", StatusLedPin)

	oshi.actionLed, e = hwio.GetPinWithMode(ActionLedPin, hwio.OUTPUT)
	if e != nil {
		panic(e)
	}
	log.Printf("Set pin %s as actionLed\n", ActionLedPin)
}

func readTracker(name string, TrackerPin hwio.Pin) {

	oldValue := 0            //value readed from tracker, initially set to 0, because the tracker was innactive
	timeAction := time.Now() // time of the action detected

	// loop
	for theContext.State != STOPPED {
		// Read the tracker value
		value, e := hwio.DigitalRead(TrackerPin)
		if e != nil {
			panic(e)
		}
		//timeActionOld=timeAction //store the last time
		timeAction = time.Now() // time at this point
		// Did value change?
		if (value == 1) && (value != oldValue) {
			dataString := fmt.Sprintf("[%s], %d,\n",
				name, int64(timeAction.Sub(theContext.getTime0())/time.Microsecond))
			log.Println(dataString)
			theContext.DataFile.WriteString(dataString)

			// Write the value to the led indicating somewhat is happened
			if value == 1 {
				hwio.DigitalWrite(theOshi.actionLed, hwio.HIGH)
			} else {
				hwio.DigitalWrite(theOshi.actionLed, hwio.LOW)
			}
		}
		oldValue = value
	}
}

func (cntxt *Context) readFromArduino() {

	var register, reg []byte
	// operate with the gobal variables theSensorData and theSensorDataInBytes; more speed?

	// don't use the first readding ??  I'm not sure about that
	reader := bufio.NewReader(cntxt.SerialPort)
	// find the begging of an stream of data from the sensors
	register, err := reader.ReadBytes('\x24')
	if err != nil {
		log.Fatal(err)
	}
	//log.Println(register)
	//log.Printf(">>>>>>>>>>>>>>")

	// loop
	for cntxt.State != STOPPED {
		// Read the serial and decode

		register = nil
		reg = nil

		//n, err = s.Read(register)
		for len(register) < 38 { // in case of \x24 chars repeted the length will be less than the expected 38 bytes
			reg, err = reader.ReadBytes('\x24')
			if err != nil {
				log.Fatal(err)
			}
			register = append(register, reg...)
		}

		receptionTime := time.Now() // time of the action detected

		if register[0] == '\x23' { // if first byte is '#', lets decode the stream of bytes in register

			//decode the register

			theSensorDataInBytes.sincroMicroSecondsInBytes = register[1:5]
			buf := bytes.NewReader(theSensorDataInBytes.sincroMicroSecondsInBytes)
			err = binary.Read(buf, binary.LittleEndian, &theSensorData.sincroMicroSeconds)

			theSensorDataInBytes.sensorMicroSecondsInBytes = register[5:9]
			buf = bytes.NewReader(theSensorDataInBytes.sensorMicroSecondsInBytes)
			err = binary.Read(buf, binary.LittleEndian, &theSensorData.sensorMicroSeconds)

			theSensorDataInBytes.distanceInBytes = register[9:13]
			buf = bytes.NewReader(theSensorDataInBytes.distanceInBytes)
			err = binary.Read(buf, binary.LittleEndian, &theSensorData.distance)

			theSensorDataInBytes.accXInBytes = register[13:17]
			buf = bytes.NewReader(theSensorDataInBytes.accXInBytes)
			err = binary.Read(buf, binary.LittleEndian, &theSensorData.accX)

			theSensorDataInBytes.accYInBytes = register[17:21]
			buf = bytes.NewReader(theSensorDataInBytes.accYInBytes)
			err = binary.Read(buf, binary.LittleEndian, &theSensorData.accY)

			theSensorDataInBytes.accZInBytes = register[21:25]
			buf = bytes.NewReader(theSensorDataInBytes.accZInBytes)
			err = binary.Read(buf, binary.LittleEndian, &theSensorData.accZ)

			theSensorDataInBytes.gyrXInBytes = register[25:29]
			buf = bytes.NewReader(theSensorDataInBytes.gyrXInBytes)
			err = binary.Read(buf, binary.LittleEndian, &theSensorData.gyrX)

			theSensorDataInBytes.gyrYInBytes = register[29:33]
			buf = bytes.NewReader(theSensorDataInBytes.gyrYInBytes)
			err = binary.Read(buf, binary.LittleEndian, &theSensorData.gyrY)

			theSensorDataInBytes.gyrZInBytes = register[33:37]
			buf = bytes.NewReader(theSensorDataInBytes.gyrZInBytes)
			err = binary.Read(buf, binary.LittleEndian, &theSensorData.gyrZ)

		} // if

		//compound the dataline and write to the output
		//receptionTime= time.Now() // Alternative: time at this point
		dataString := fmt.Sprintf("[%s], %d, %d, %d, %d, %f, %f, %f, %f, %f, %f\n", "Ard", int64(receptionTime.Sub(cntxt.Time0)/time.Microsecond), theSensorData.sincroMicroSeconds, theSensorData.sensorMicroSeconds, theSensorData.distance, theSensorData.accX, theSensorData.accY, theSensorData.accZ, theSensorData.gyrX, theSensorData.gyrY, theSensorData.gyrZ)

		log.Println(dataString)
		cntxt.DataFile.WriteString(dataString)
		// Write the value to the led indicating somewhat is happened
		hwio.DigitalWrite(theOshi.actionLed, hwio.HIGH)
		hwio.DigitalWrite(theOshi.actionLed, hwio.LOW)
	}
}

// // another version looking for more speed, based in local variables
// func (acq *Acquisition) readFromArduino2() {
//
// 	var register, reg []byte
// 	var sincroMicroSecondsInBytes []byte
// 	var sensorMicroSecondsInBytes []byte
// 	var distanceInBytes []byte
// 	var accXInBytes []byte
// 	var accYInBytes []byte
// 	var accZInBytes []byte
// 	var gyrXInBytes []byte
// 	var gyrYInBytes []byte
// 	var gyrZInBytes []byte
// 	var sincroMicroSeconds uint32
// 	var sensorMicroSeconds uint32
// 	var distance uint32
// 	var accX float32
// 	var accY float32
// 	var accZ float32
// 	var gyrX float32
// 	var gyrY float32
// 	var gyrZ float32
//
// 	// don't use the first readding ??  I'm not sure about that
// 	reader := bufio.NewReader(acq.serialPort)
// 	// find the begging of an stream of data from the sensors
// 	register, err := reader.ReadBytes('\x24')
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	//log.Println(register)
// 	//log.Printf(">>>>>>>>>>>>>>")
//
// 	// loop
// 	for acq.getState() != stateSTOPPED {
// 		// Read the serial and decode
//
// 		register = nil
// 		reg = nil
//
// 		//n, err = s.Read(register)
// 		for len(register) < 38 { // in case of \x24 chars repeted maked the length will be less than the expected 38 bytes
// 			reg, err = reader.ReadBytes('\x24')
// 			if err != nil {
// 				log.Fatal(err)
// 			}
// 			register = append(register, reg...)
// 		}
//
// 		receptionTime := time.Now() // time of the action detected
//
// 		if register[0] == '\x23' {
//
// 			//decode the register
//
// 			sincroMicroSecondsInBytes = register[1:5]
// 			buf := bytes.NewReader(sincroMicroSecondsInBytes)
// 			err = binary.Read(buf, binary.LittleEndian, &sincroMicroSeconds)
//
// 			sensorMicroSecondsInBytes = register[5:9]
// 			buf = bytes.NewReader(sensorMicroSecondsInBytes)
// 			err = binary.Read(buf, binary.LittleEndian, &sensorMicroSeconds)
//
// 			distanceInBytes = register[9:13]
// 			buf = bytes.NewReader(distanceInBytes)
// 			err = binary.Read(buf, binary.LittleEndian, &distance)
//
// 			accXInBytes = register[13:17]
// 			buf = bytes.NewReader(accXInBytes)
// 			err = binary.Read(buf, binary.LittleEndian, &accX)
//
// 			accYInBytes = register[17:21]
// 			buf = bytes.NewReader(accYInBytes)
// 			err = binary.Read(buf, binary.LittleEndian, &accY)
//
// 			accZInBytes = register[21:25]
// 			buf = bytes.NewReader(accZInBytes)
// 			err = binary.Read(buf, binary.LittleEndian, &accZ)
//
// 			gyrXInBytes = register[25:29]
// 			buf = bytes.NewReader(gyrXInBytes)
// 			err = binary.Read(buf, binary.LittleEndian, &gyrX)
//
// 			gyrYInBytes = register[29:33]
// 			buf = bytes.NewReader(gyrYInBytes)
// 			err = binary.Read(buf, binary.LittleEndian, &gyrY)
//
// 			gyrZInBytes = register[33:37]
// 			buf = bytes.NewReader(gyrZInBytes)
// 			err = binary.Read(buf, binary.LittleEndian, &gyrZ)
//
// 		} // if
//
// 		if acq.getState() != statePAUSED {
// 			//compound the dataline and write to the output
// 			//receptionTime= time.Now() // Alternative: time at this point
// 			dataString := fmt.Sprintf("[%s], %v, %d, %d, %d, %f, %f, %f, %f, %f, %f\n",
// 				"Ard", receptionTime.Sub(theAcq.getTime0()),
// 				sincroMicroSeconds, sensorMicroSeconds, distance,
// 				accX, accY, accZ, gyrX, gyrY, gyrZ)
//
// 			log.Println(dataString)
// 			acq.outputFile.WriteString(dataString)
// 		}
// 		// Write the value to the led indicating somewhat is happened
// 		hwio.DigitalWrite(theOshi.actionLed, hwio.HIGH)
// 		hwio.DigitalWrite(theOshi.actionLed, hwio.LOW)
// 	}
// }

func blinkingLed(ledPin hwio.Pin) int {
	// loop
	for {
		hwio.DigitalWrite(ledPin, hwio.HIGH)
		hwio.Delay(500)
		hwio.DigitalWrite(ledPin, hwio.LOW)
		hwio.Delay(500)
	}
}

func waitTillButtonPushed(buttonPin hwio.Pin) int {

	// loop
	for {
		// Read the tracker value
		value, e := hwio.DigitalRead(buttonPin)
		if e != nil {
			panic(e)
		}
		// Was the button pressed, value = 1?
		if value == 1 {
			return value
		}
	}
}

//////////////
// Web section (web server prototype not connected)
//////////////

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
	fmt.Println(">>>", theContext)

	theContext.Title = titleWelcome
	render(w, "index", theContext)
}

//ThePlatform describes the system
func ThePlatform(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", theContext)

	theContext.Message = "Description of the Platform"
	theContext.AlertLevel = INFO
	theContext.Title = titleThePlatform
	render(w, "thePlatform", theContext)
}

//Init set the platform in a initial state
func Init(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", theContext)

	switch theContext.State {
	case INIT, CONFIGURED, STOPPED:
		// correct states
		if req.Method == "GET" {
			theContext.Message = "Warning! You are erasing the configuration, the datafiles and restoring the platform to it's initial state."
			theContext.AlertLevel = DANGER
			theContext.Title = titleInit
			render(w, "init", theContext)
		} else { // POST
			fmt.Println("POST")
			req.ParseForm()
			fmt.Println(req.Form)
			if req.Form.Get("initializate") == "YES" {
				//if YES, init the platform
				theContext.State = INIT
				theContext.ConfigurationName = ""
				//erase datafiles
				dataDirectory := filepath.Join(StaticRoot, DataFilePath)
				fmt.Println("DELETING ", dataDirectory)
				err := RemoveContents(dataDirectory)
				if err != nil {
					fmt.Println(err)
				}

				//message of initial state
				theContext.Message = "The system is now in the initial state. Now you must define a new configuration berofe run an experiment."
				theContext.AlertLevel = SUCCESS
			} else {
				//message of initial state
				theContext.Message = "The system initialization is canceled. The current configuration is active."
				theContext.AlertLevel = WARNING
			}
			//initiated or not, shows the experiment page
			theContext.Title = titleExperiment
			render(w, "experiment", theContext)
		}
	case RUNNING:
		// wrong state
		theContext.Message = "System is running! It MUST be stopped before erase the configuration and set the initial state."
		theContext.AlertLevel = DANGER
		theContext.Title = titleRun
		render(w, "run", theContext)
	}

}

//Experiment allows to access to the experiments
func Experiment(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", theContext)

	switch theContext.State {
	case INIT, CONFIGURED, STOPPED:
		//correct cases, shows the experiment page to config,test and run it
		theContext.Message = "Let's make some experiments"
		theContext.AlertLevel = INFO
		theContext.Title = titleExperiment
		render(w, "experiment", theContext)
	case RUNNING:
		//wrong case, it must be STOPPED before
		theContext.Message = "System is running! It MUST be stopped before a new configuration done."
		theContext.AlertLevel = DANGER
		theContext.Title = titleRun
		render(w, "run", theContext)
	}
}

//Config allows to configure the sensors
func Config(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", theContext)

	switch theContext.State {
	case INIT, CONFIGURED, STOPPED:
		//correct states, do the config process
		if req.Method == "GET" {
			theContext.Message = "Activate/Deactivate the sensors."
			theContext.AlertLevel = INFO
			theContext.Title = titleConfig
			render(w, "config", theContext)
		} else { // POST
			fmt.Println("POST")
			req.ParseForm()
			// logic part of login
			//validation phase will be here
			//if valid, put the form data into the context struct
			theContext.ConfigurationName = req.Form.Get("ConfigurationName")
			if req.Form.Get("SetTrackerA") == SensorStateOn {
				theContext.SetTrackerA = ON
			} else {
				theContext.SetTrackerA = OFF
			}
			if req.Form.Get("SetTrackerB") == SensorStateOn {
				theContext.SetTrackerB = ON
			} else {
				theContext.SetTrackerB = OFF
			}
			if req.Form.Get("SetTrackerC") == SensorStateOn {
				theContext.SetTrackerC = ON
			} else {
				theContext.SetTrackerC = OFF
			}
			if req.Form.Get("SetTrackerD") == SensorStateOn {
				theContext.SetTrackerD = ON
			} else {
				theContext.SetTrackerD = OFF
			}
			if req.Form.Get("SetTrackerM") == SensorStateOn {
				theContext.SetTrackerM = ON
			} else {
				theContext.SetTrackerM = OFF
			}
			if req.Form.Get("SetDistance") == SensorStateOn {
				theContext.SetDistance = ON
			} else {
				theContext.SetDistance = OFF
			}
			if req.Form.Get("SetAccGyro") == SensorStateOn {
				theContext.SetAccGyro = ON
			} else {
				theContext.SetAccGyro = OFF
			}
			//prepare the context
			theContext.Message = "Configuration done! Now the system can be tested or runned the experiment"
			theContext.AlertLevel = SUCCESS
			theContext.State = CONFIGURED
			theContext.Title = titleExperiment
			//log
			fmt.Println(req.Form)
			fmt.Println("Contex:", theContext)
			//once processed the form, reditect to the index page

			//render(w, "experiment", theContext)
			http.Redirect(w, req, "/experiment/", http.StatusFound)
		}
	case RUNNING:
		// only put a message, but don't touch the running process
		theContext.Message = "System is running! It MUST be stopped before a new configuration done."
		theContext.AlertLevel = DANGER
		theContext.Title = titleRun
		render(w, "run", theContext)
	}
}

//Test allows to test the sensors
func Test(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", theContext)

	switch theContext.State {
	case INIT:
		//The system must be configured before
		theContext.Message = "The system must be configured before you could test it!"
		theContext.AlertLevel = WARNING
		theContext.Title = titleConfig
		render(w, "configure", theContext)
	case RUNNING:
		//wrong state, the system must be stopped before
		theContext.Message = "Warning! You must stop the system before test the system."
		theContext.AlertLevel = DANGER
		theContext.Title = titleRun
		render(w, "run", theContext)
	case CONFIGURED, STOPPED:
		//correct state, let's test the system, and then to experiment page

		//check state of the sensors and put it on stateOfSensors
		//put here the test code
		//put here the test code
		//put here the test code
		//put here the test code

		// this test is a naive one, only for demonstration purpose
		if theContext.SetTrackerA {
			theContext.StateOfTrackerA = READY
		} else {
			theContext.StateOfTrackerA = DISSABLED
		}
		if theContext.SetTrackerB {
			theContext.StateOfTrackerB = READY
		} else {
			theContext.StateOfTrackerB = DISSABLED
		}
		if theContext.SetTrackerC {
			theContext.StateOfTrackerC = READY
		} else {
			theContext.StateOfTrackerC = DISSABLED
		}
		if theContext.SetTrackerD {
			theContext.StateOfTrackerD = READY
		} else {
			theContext.StateOfTrackerD = DISSABLED
		}
		if theContext.SetTrackerM {
			theContext.StateOfTrackerM = READY
		} else {
			theContext.StateOfTrackerM = DISSABLED
		}
		if theContext.SetDistance {
			theContext.StateOfDistance = READY
		} else {
			theContext.StateOfDistance = DISSABLED
		}
		if theContext.SetAccGyro {
			theContext.StateOfAccGyro = READY
		} else {
			theContext.StateOfAccGyro = DISSABLED
		}
		// test done, shows the result

		theContext.Title = titleTest
		theContext.Message = "System configured and Tested. Ready to run."
		theContext.AlertLevel = SUCCESS
		fmt.Println(">>>", theContext)
		render(w, "test", theContext)
	}
}

//Run allows to run the experiments
func Run(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", theContext)

	switch theContext.State {
	case INIT:
		//wrong state, show experiment page
		theContext.Message = "Warning! You must configure the system before run the experiment."
		theContext.AlertLevel = DANGER
		theContext.Title = titleExperiment
		//http.Redirect(w, req, "/experiment/", http.StatusFound)
		render(w, "experiment", theContext)
	case CONFIGURED, STOPPED:
		//correct states, do the running process

		dataFileName := filepath.Join(StaticRoot, DataFilePath, theContext.ConfigurationName+DataFileExtension)
		//detect if file exists
		_, err := os.Stat(dataFileName)
		//create datafile is not exists
		if os.IsNotExist(err) {
			//create file to write
			fmt.Println("Creating ", dataFileName)
			theContext.DataFile, err = os.Create(dataFileName)
			if err != nil {
				fmt.Println(err.Error())
			}
			statusLine := fmt.Sprintf("### %v Data Acquisition: %s \n\n", time.Now(), theContext.ConfigurationName)
			theContext.DataFile.WriteString(statusLine)
			formatLine := fmt.Sprintf("### [Ard], localTime(us), sincroTime(us), sensorTime(us), distance(mm), accX(g), accY(g), accZ(g), gyrX(gr/s), gyrY(gr/s), gyrZ(gr/s) \n\n")
			theContext.DataFile.WriteString(formatLine)
			// sets the new time0 only with a new scenery
			theContext.setTime0()
		} else {
			//open fle to append
			fmt.Println("Openning ", dataFileName)
			theContext.DataFile, err = os.OpenFile(dataFileName, os.O_RDWR|os.O_APPEND, 0644)
			if err != nil {
				fmt.Println(err.Error())
			}
		}

		// running process instruction here!
		// running process instruction here!

		//waitTillButtonPushed(buttonA)
		hwio.DigitalWrite(theOshi.statusLed, hwio.HIGH)
		log.Println("Beginning.....")

		// launch the trackers

		log.Printf("There are %v goroutines", runtime.NumGoroutine())
		log.Printf("Launching the Gourutines")

		go theContext.readFromArduino()
		log.Println("Started Arduino")
		go readTracker("A", theOshi.trackerA)
		log.Println("Started Tracker A")
		go readTracker("B", theOshi.trackerB)
		log.Println("Started Tracker B")
		go readTracker("C", theOshi.trackerC)
		log.Println("Started Tracker C")
		go readTracker("D", theOshi.trackerD)
		log.Println("Started Tracker D")

		log.Printf("There are %v goroutines", runtime.NumGoroutine())
		//dump the data gathered in DataFile
		//_, err = context.DataFile.WriteString("123, 123, 123, 123\n")
		//err = context.DataFile.Sync()
		//if err != nil {
		//	fmt.Println(err.Error())
		//}
		//defer close the file to STOP

		theContext.Message = "System running gathering data from sensors."
		theContext.AlertLevel = SUCCESS
		theContext.Title = titleRun
		theContext.State = RUNNING
		render(w, "run", theContext)
	case RUNNING:
		// we already are in this State
		// only put a message, but don't touch the running process
		theContext.Message = "System is ALREADY running!"
		theContext.AlertLevel = WARNING
		theContext.Title = titleRun
		render(w, "run", theContext)
	}
}

//Stop allows to stop the experiments
func Stop(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", theContext)

	switch theContext.State {
	case INIT, CONFIGURED:
		theContext.Message = "Warning! You must configure the system and run the experiment before stop it."
		theContext.AlertLevel = DANGER
		theContext.Title = titleExperiment
		render(w, "experiment", theContext)
	case RUNNING:
		//correct state, do the stop process
		// stop process instruction here!
		// stop process instruction here!
		// stop process instruction here!
		hwio.DigitalWrite(theOshi.statusLed, hwio.LOW)
		// close the GPIO pins
		//hwio.CloseAll()

		//close the file
		err := theContext.DataFile.Sync()
		if err != nil {
			fmt.Println(err.Error())
		}
		theContext.DataFile.Close()

		theContext.Title = titleStop
		theContext.State = STOPPED
		theContext.Message = "System stopped. Now you can donwload the data to your permanent storage"
		theContext.AlertLevel = SUCCESS
		render(w, "stop", theContext)

	case STOPPED:
		// we already are in this State
		// only put a message, but don't touch the process
		theContext.Message = "System is ALREADY stooped!"
		theContext.AlertLevel = WARNING
		theContext.Title = titleStop
		render(w, "experiment", theContext)
	}
}

//Collect the data gathered in the experiments
func Collect(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", theContext)

	switch theContext.State {
	case INIT, CONFIGURED, STOPPED:
		//read the data directory and offers the files to be downloaded
		theContext.DataFiles, _ = filepath.Glob(filepath.Join(StaticRoot, DataFilePath, "*"+DataFileExtension))
		//fmt.Println(">>>> " + filepath.Join(StaticRoot, DataFilePath, "*"+DataExtension))
		//let only the file name, eliminate the path
		for i, f := range theContext.DataFiles {
			theContext.DataFiles[i] = path.Base(f)
		}

		fmt.Println(theContext.DataFiles)

		theContext.Title = titleCollect
		if len(theContext.DataFiles) == 0 {
			theContext.Message = "Sorry! There are not files to donwload stored in the system."
			theContext.AlertLevel = WARNING
		} else {
			theContext.Message = "You can download the data stored in the system."
			theContext.AlertLevel = INFO
		}
		render(w, "collect", theContext)
	case RUNNING:
		theContext.Message = "You can't download data is while the system is running. You must stop the system before."
		theContext.AlertLevel = WARNING
		theContext.Title = titleRun
		render(w, "run", theContext)
	}

}

//About shows the page with info
func About(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", theContext)

	theContext.Title = titleAbout
	render(w, "about", theContext)
}

//Help shows information about the tool
func Help(w http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>", req.URL)
	fmt.Println(">>>", theContext)

	theContext.Title = titleHelp
	render(w, "help", theContext)
}

// render
func render(w http.ResponseWriter, tmpl string, cntxt Context) {
	fmt.Println("[render]>>>", cntxt)
	cntxt.Static = StaticURL
	//list of templates, put here all the templates needed
	tmplList := []string{"templates/base.html",
		fmt.Sprintf("templates/message.html"),
		fmt.Sprintf("templates/%s.html", tmpl)}
	t, err := template.ParseFiles(tmplList...)
	if err != nil {
		log.Print("template parsing error: ", err)
	}
	err = t.Execute(w, cntxt)
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
	theContext.State = INIT
	theContext.initiate()
	theOshi.initiate()

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

	// close the GPIO pins
	defer theContext.SerialPort.Close()
	hwio.CloseAll()
}
