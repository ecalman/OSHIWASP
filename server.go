package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
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
	ActionLedPin = "gpio8" // yellow or red

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

//state of system
const (
	INIT       = 0
	CONFIGURED = 1
	RUNNING    = 2
	STOPPED    = 3
	POWEROFF   = -1
)

//language
const (
	nLangs  = 2
	ENGLISH = 0
	SPANISH = 1
)

//title of pages respect of state
var (
	titleWelcome     [nLangs]string
	titleThePlatform [nLangs]string
	titleInit        [nLangs]string
	titleConfig      [nLangs]string
	titleTest        [nLangs]string
	titleExperiment  [nLangs]string
	titleRun         [nLangs]string
	titleStop        [nLangs]string
	titleCollect     [nLangs]string
	titlePoweroff    [nLangs]string
	titleAbout       [nLangs]string
	titleHelp        [nLangs]string
	titleTheEnd      [nLangs]string
)

//messages of the pages
var (
	messageThePlatform        [nLangs]string
	messageInitICSGet         [nLangs]string
	messageInitICSPostYes     [nLangs]string
	messageInitICSPostNo      [nLangs]string
	messageInitR              [nLangs]string
	messageExperimentICS      [nLangs]string
	messageExperimentR        [nLangs]string
	messageConfigICSGet       [nLangs]string
	messageConfigICSPost      [nLangs]string
	messageConfigR            [nLangs]string
	messageTestI              [nLangs]string
	messageTestR              [nLangs]string
	messageTestCS             [nLangs]string
	messageRunI               [nLangs]string
	messageRunR               [nLangs]string
	messageRunCS              [nLangs]string
	messageStopIC             [nLangs]string
	messageStopR              [nLangs]string
	messageStopS              [nLangs]string
	messageCollectICS0        [nLangs]string
	messageCollectICS         [nLangs]string
	messageCollectR           [nLangs]string
	messagePoweroffICSGet     [nLangs]string
	messagePoweroffICSPostYes [nLangs]string
	messagePoweroffICSPostNo  [nLangs]string
	messagePoweroffR          [nLangs]string
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
	//language
	Lang int

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
	SetTrackerA      bool
	SetTrackerB      bool
	SetTrackerC      bool
	SetTrackerD      bool
	SetTrackerM      bool
	SetDistance      bool
	SetAccelerometer bool
	SetGyroscope     bool
	// state of sensor after test calling
	StateOfTrackerA      int
	StateOfTrackerB      int
	StateOfTrackerC      int
	StateOfTrackerD      int
	StateOfTrackerM      int
	StateOfDistance      int
	StateOfAccelerometer int
	StateOfGyroscope     int
}

// SensorDataInBytes data for sensors in Arduino in bytes
type SensorDataInBytes struct {
	trackerMicroSecondsInBytes []byte
	sensorMicroSecondsInBytes  []byte
	distanceInBytes            []byte
	accXInBytes                []byte
	accYInBytes                []byte
	accZInBytes                []byte
	gyrXInBytes                []byte
	gyrYInBytes                []byte
	gyrZInBytes                []byte
}

// SensorData data for sensors in Arduino in numerical data types
type SensorData struct {
	trackerMicroSeconds uint32
	sensorMicroSeconds  uint32
	distance            uint32
	accX                float32
	accY                float32
	accZ                float32
	gyrX                float32
	gyrY                float32
	gyrZ                float32
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
	var err error
	// config the comm port for serial via BT
	commPort := &serial.Config{Name: CommDevName, Baud: Bauds}
	// open the serial comm with the arduino via BT
	cntxt.SerialPort, err = serial.OpenPort(commPort)
	if err != nil {
		log.Printf("error opening the serial port with Arduino")
		log.Fatal(err)
	}
	//defer acq.serialPort.Close()
	log.Printf("Open serial device %s", CommDevName)
}

func setArduinoStateON() {
	// activate the readdings in Arduino sending 'ON'
	log.Printf("before write on")
	_, err := theContext.SerialPort.Write([]byte("n"))
	log.Printf("after write on")
	if err != nil {
		log.Fatal(err)
	}
}

func setArduinoStateOFF() {
	// deactivate the readdings in Artudino sending 'OFF'
	log.Printf("before write off")
	_, err := theContext.SerialPort.Write([]byte("f"))
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

func (cntxt *Context) createOutputFile() {
	var e error
	cntxt.DataFileName = DataFilePath + cntxt.ConfigurationName + DataFileExtension
	cntxt.DataFile, e = os.Create(cntxt.DataFileName)
	if e != nil {
		panic(e)
	}
	statusLine := fmt.Sprintf("### %v Data Acquisition: %s \n\n", time.Now(), cntxt.ConfigurationName)
	cntxt.DataFile.WriteString(statusLine)
	formatLine := fmt.Sprintf("### [Ard], localTime(us), sensorTime(us)")
	if cntxt.SetTrackerM == ON {
		formatLine += fmt.Sprintf(", trackerTime(us)")
	}
	if cntxt.SetDistance == ON {
		formatLine += fmt.Sprintf(", distance(mm)")
	}
	if cntxt.SetAccelerometer == ON {
		formatLine += fmt.Sprintf(", accX(g)")
		formatLine += fmt.Sprintf(", accY(g)")
		formatLine += fmt.Sprintf(", accZ(g)")
	}
	if cntxt.SetGyroscope == ON {
		formatLine += fmt.Sprintf(", gyrX(gr/s)")
		formatLine += fmt.Sprintf(", gyrY(gr/s)")
		formatLine += fmt.Sprintf(", gyrZ(gr/s)")
	}
	formatLine += fmt.Sprintf("\n\n")
	cntxt.DataFile.WriteString(formatLine)

	log.Printf("Cretated output File %s", cntxt.DataFileName)
}

func (cntxt *Context) initiate() {

	//set language
	cntxt.Lang = SPANISH

	//set the titles of the pages

	titleWelcome[ENGLISH] = "Welcome!"
	titleWelcome[SPANISH] = "Bienvenidos!"
	titleThePlatform[ENGLISH] = "The Platform"
	titleThePlatform[SPANISH] = "La Plataforma"
	titleInit[ENGLISH] = "Initialization"
	titleInit[SPANISH] = "Inicialización"
	titleConfig[ENGLISH] = "Configuration of Sensor Platform"
	titleConfig[SPANISH] = "Configuración de la Plataforma de Sensores"
	titleTest[ENGLISH] = "Test the Sensor Platform"
	titleTest[SPANISH] = "Prueba la Plataforma de Sensores"
	titleExperiment[ENGLISH] = "Experiment"
	titleExperiment[SPANISH] = "Experimento"
	titleRun[ENGLISH] = "Run"
	titleRun[SPANISH] = "Ejecución"
	titleStop[ENGLISH] = "Stop"
	titleStop[SPANISH] = "Parada"
	titleCollect[ENGLISH] = "Collect Data"
	titleCollect[SPANISH] = "Recopilar los Datos"
	titlePoweroff[ENGLISH] = "Power off"
	titlePoweroff[SPANISH] = "Apagar"
	titleAbout[ENGLISH] = "About"
	titleAbout[SPANISH] = "Sobre mi"
	titleHelp[ENGLISH] = "Help"
	titleHelp[SPANISH] = "Ayuda"
	titleTheEnd[ENGLISH] = "The End"
	titleTheEnd[SPANISH] = "Fin"

	//set the messages of the pages
	messageThePlatform[ENGLISH] = "Description of the Platform"
	messageThePlatform[SPANISH] = "Descripción de la Plataforma"
	messageInitICSGet[ENGLISH] = "Warning! You are erasing the configuration, the datafiles and restoring the platform to it's initial state."
	messageInitICSGet[SPANISH] = "Atención! Está borrando la configuración, los archivos con los datos y restaurando la plataforma a su estado inicial."
	messageInitICSPostYes[ENGLISH] = "The platform is now in the initial state. Now you must define a new configuration berofe run an experiment."
	messageInitICSPostYes[SPANISH] = "La plataforma ahora está en su estado inicial. Debe definir una nueva configuración antres de ejecutar un experimento."
	messageInitICSPostNo[ENGLISH] = "The platform initialization is canceled. The current configuration is active."
	messageInitICSPostNo[SPANISH] = "Inicialización de la plataforma cancelada. La configuración actual sigue activa."
	messageInitR[ENGLISH] = "An experiment is running! It MUST be stopped before erase the configuration and set the initial state."
	messageInitR[SPANISH] = "Un experimento está en ejecución! DEBE pararse antes de borrar la configuración y reestablecer el estado inicial."
	messageExperimentICS[ENGLISH] = "Let's make some experiments"
	messageExperimentICS[SPANISH] = "Hagamos algunos experimentos"
	messageExperimentR[ENGLISH] = "An experiment is already running! It MUST be stopped before a new experiment could be run."
	messageExperimentR[SPANISH] = "Un experimento ya está en ejecución! DEBE ser parado antes de ejecutar otro."
	messageConfigICSGet[ENGLISH] = "Activate/Deactivate the sensors."
	messageConfigICSGet[SPANISH] = "Activar/Desactivar los sensores."
	messageConfigICSPost[ENGLISH] = "Configuration done! Now the platform can be tested or runned the experiment"
	messageConfigICSPost[SPANISH] = "Configuración hecha! Ahora puede comprobar la plataforma o ejecutar el experimento"
	messageConfigR[ENGLISH] = "Experiment is running! It MUST be stopped before a new configuration done."
	messageConfigR[SPANISH] = "Experimento en ejecución! Debe ser parado antes de fijar una configuración nueva."
	messageTestI[ENGLISH] = "The platform must be configured before you could test it!"
	messageTestI[SPANISH] = "La plataforma debe ser configurada antes de que pueda ser comprobada!"
	messageTestR[ENGLISH] = "Warning! You must stop the experimento before test the system."
	messageTestR[SPANISH] = "Atención! Debe parar el experimento antes de poder comprobar la plataforma."
	messageTestCS[ENGLISH] = "Sorry, but not implemented yet!. Ready to run."
	messageTestCS[SPANISH] = "Discupas, pero aún no está implementado! Listo para ejecutar."
	messageRunI[ENGLISH] = "Warning! You must configure the system before run the experiment."
	messageRunI[SPANISH] = "Atención! Debe Configurar la platraforma antes de poder ejecutar un experimento."
	messageRunR[ENGLISH] = "Experiment is ALREADY running!"
	messageRunR[SPANISH] = "Experimento YA en ejecución!"
	messageRunCS[ENGLISH] = "Experiment running and gathering data from sensors."
	messageRunCS[SPANISH] = "Experimento en ejecución y adquiriendo datos de los sensoresción y adquiriendo datos de los sensores."
	messageStopIC[ENGLISH] = "Warning! You must configure the platform and run the experiment before stop it."
	messageStopIC[SPANISH] = "Atención! Debe configurar y ejecutar el experimento antes de poder pararlo."
	messageStopR[ENGLISH] = "Experiment stopped. Now you can donwload the data to your permanent storage"
	messageStopR[SPANISH] = "Experimento parado. Ahora puede descargar los datos a su almacenamiento permanente"
	messageStopS[ENGLISH] = "The experiment is ALREADY stooped!"
	messageStopS[SPANISH] = "El experimento YA está parado!"
	messageCollectICS0[ENGLISH] = "Sorry! There is not any file with data stored in the system."
	messageCollectICS0[SPANISH] = "Disculpe, pero no hay ningún archivo con datos almacenado en el sistema."
	messageCollectICS[ENGLISH] = "You can download the data stored in the system."
	messageCollectICS[SPANISH] = "Puede descargar los datos almacenados en el sistema."
	messageCollectR[ENGLISH] = "You can't download data while the experiment is running. You must stop it before."
	messageCollectR[SPANISH] = "No se pueden recoger datos mientas el experimento está en ejecución. Debe pararlo antes."

	messagePoweroffICSGet[ENGLISH] = "Warning! You are switching the system off."
	messagePoweroffICSGet[SPANISH] = "Atención! Va a proceder a apagar el sistema."
	messagePoweroffICSPostYes[ENGLISH] = "The system is now POWERING OFF. Wait a moment until all the activity stops."
	messagePoweroffICSPostYes[SPANISH] = "El sistema se esta APAGANDO. Espere un momento a que toda la actividad cese."
	messagePoweroffICSPostNo[ENGLISH] = "The system power off is canceled. The current configuration is active."
	messagePoweroffICSPostNo[SPANISH] = "El apagado del sistema ha sido cancelado. La configuración actual sige activa."
	messagePoweroffR[ENGLISH] = "The experiment is running! It MUST be stopped before switch the system off."
	messagePoweroffR[SPANISH] = "El experimento está en ejecución! Debe ser parado antes de apagar el sistema."

	//acq.setOutputFileName(dataPath+dataFileName+dataFileExtension)
	//acq.createOutputFile()
	cntxt.connectArduinoSerialBT()
	log.Printf("Arduino connected!")
	//cntxt.setStateNEW()
	cntxt.State = INIT
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

	//value readed from tracker, initially set to 0, because the tracker was innactive
	oldValue := 0
	// time of the action detected
	timeAction := time.Now()

	// loop
	for theContext.State == RUNNING {
		// Read the tracker value
		value, e := hwio.DigitalRead(TrackerPin)
		if e != nil {
			panic(e)
		}
		//timeActionOld=timeAction //store the last time
		timeAction = time.Now() // time at this point
		// Did value change?
		if (value == 1) && (value != oldValue) {
			dataString := fmt.Sprintf("[%s]; %d\n",
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
		log.Println(err)
	}
	//log.Println(register)
	//log.Printf(">>>>>>>>>>>>>>")

	// loop
	for cntxt.State == RUNNING {
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

			theSensorDataInBytes.trackerMicroSecondsInBytes = register[1:5]
			buf := bytes.NewReader(theSensorDataInBytes.trackerMicroSecondsInBytes)
			err = binary.Read(buf, binary.LittleEndian, &theSensorData.trackerMicroSeconds)

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
		dataString := fmt.Sprintf("[%s]; %d; %d", "Ard",
			int64(receptionTime.Sub(cntxt.Time0)/time.Microsecond), theSensorData.sensorMicroSeconds)
		if cntxt.SetTrackerM == ON {
			dataString += fmt.Sprintf("; %d", theSensorData.trackerMicroSeconds)
		}
		if cntxt.SetDistance == ON {
			dataString += fmt.Sprintf("; %d", theSensorData.distance)
		}
		if cntxt.SetAccelerometer == ON {
			dataString += fmt.Sprintf("; %f; %f; %f",
				theSensorData.accX, theSensorData.accY, theSensorData.accZ)
		}
		if cntxt.SetGyroscope == ON {
			dataString += fmt.Sprintf("; %f; %f; %f",
				theSensorData.gyrX, theSensorData.gyrY, theSensorData.gyrZ)
		}
		dataString += "\n" //end of line

		log.Println(dataString)
		cntxt.DataFile.WriteString(dataString)
		// Write the value to the led indicating somewhat is happened
		hwio.DigitalWrite(theOshi.actionLed, hwio.HIGH)
		hwio.DigitalWrite(theOshi.actionLed, hwio.LOW)
	}
}

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
// Web section
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
	log.Println(">>>", req.URL)
	log.Println(">>>", theContext)

	theContext.Title = titleWelcome[theContext.Lang]
	render(w, "index", theContext)
}

//ThePlatform describes the system
func ThePlatform(w http.ResponseWriter, req *http.Request) {
	log.Println(">>>", req.URL)
	log.Println(">>>", theContext)

	theContext.Message = messageThePlatform[theContext.Lang]
	theContext.AlertLevel = INFO
	theContext.Title = titleThePlatform[theContext.Lang]
	render(w, "thePlatform", theContext)
}

//Init set the platform in a initial state
func Init(w http.ResponseWriter, req *http.Request) {
	log.Println(">>>", req.URL)
	log.Println(">>>", theContext)

	switch theContext.State {
	case INIT, CONFIGURED, STOPPED:
		// correct states
		if req.Method == "GET" {
			theContext.Message = messageInitICSGet[theContext.Lang]
			theContext.AlertLevel = DANGER
			theContext.Title = titleInit[theContext.Lang]
			render(w, "init", theContext)
		} else { // POST
			log.Println("POST")
			req.ParseForm()
			log.Println(req.Form)
			if req.Form.Get("initializate") == "YES" {
				//if YES, init the platform
				theContext.State = INIT
				theContext.ConfigurationName = ""
				//set the initial state
				//theContext.initiate()
				//theOshi.initiate()
				//erase datafiles
				dataDirectory := filepath.Join(StaticRoot, DataFilePath)
				log.Println("DELETING ", dataDirectory)
				err := RemoveContents(dataDirectory)
				if err != nil {
					log.Println(err)
				}

				//message of initial state
				theContext.Message = messageInitICSPostYes[theContext.Lang]
				theContext.AlertLevel = SUCCESS
			} else {
				//message of initial state
				theContext.Message = messageInitICSPostNo[theContext.Lang]
				theContext.AlertLevel = WARNING
			}
			//initiated or not, shows the experiment page
			theContext.Title = titleExperiment[theContext.Lang]
			render(w, "experiment", theContext)
		}
	case RUNNING:
		// wrong state
		theContext.Message = messageInitR[theContext.Lang]
		theContext.AlertLevel = DANGER
		theContext.Title = titleRun[theContext.Lang]
		render(w, "run", theContext)
	}

}

//Experiment allows to access to the experiments
func Experiment(w http.ResponseWriter, req *http.Request) {
	log.Println(">>>", req.URL)
	log.Println(">>>", theContext)

	switch theContext.State {
	case INIT, CONFIGURED, STOPPED:
		//correct cases, shows the experiment page to config,test and run it
		theContext.Message = messageExperimentICS[theContext.Lang]
		theContext.AlertLevel = INFO
		theContext.Title = titleExperiment[theContext.Lang]
		render(w, "experiment", theContext)
	case RUNNING:
		//wrong case, it must be STOPPED before
		theContext.Message = messageExperimentR[theContext.Lang]
		theContext.AlertLevel = DANGER
		theContext.Title = titleRun[theContext.Lang]
		render(w, "run", theContext)
	}
}

//Config allows to configure the sensors
func Config(w http.ResponseWriter, req *http.Request) {
	log.Println(">>>", req.URL)
	log.Println(">>>", theContext)

	switch theContext.State {
	case INIT, CONFIGURED, STOPPED:
		//correct states, do the config process
		if req.Method == "GET" {
			theContext.Message = messageConfigICSGet[theContext.Lang]
			theContext.AlertLevel = INFO
			theContext.Title = titleConfig[theContext.Lang]
			render(w, "config", theContext)
		} else { // POST
			log.Println("POST")
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
			if req.Form.Get("SetAccelerometer") == SensorStateOn {
				theContext.SetAccelerometer = ON
			} else {
				theContext.SetAccelerometer = OFF
			}
			if req.Form.Get("SetGyroscope") == SensorStateOn {
				theContext.SetGyroscope = ON
			} else {
				theContext.SetGyroscope = OFF
			}
			//prepare the context
			theContext.Message = messageConfigICSPost[theContext.Lang]
			theContext.Title = titleExperiment[theContext.Lang]
			theContext.AlertLevel = SUCCESS
			theContext.State = CONFIGURED
			//setArduinoStateON() //initiate Arduino readding sensors and transfer via BT

			//log
			log.Println(req.Form)
			log.Println("Contex:", theContext)
			//once processed the form, reditect to the index page

			//render(w, "experiment", theContext)
			http.Redirect(w, req, "/experiment/", http.StatusFound)
		}
	case RUNNING:
		// only put a message, but don't touch the running process
		theContext.Message = messageConfigR[theContext.Lang]
		theContext.AlertLevel = DANGER
		theContext.Title = titleRun[theContext.Lang]
		render(w, "run", theContext)
	}
}

//Test allows to test the sensors
func Test(w http.ResponseWriter, req *http.Request) {
	log.Println(">>>", req.URL)
	log.Println(">>>", theContext)

	switch theContext.State {
	case INIT:
		//The system must be configured before
		theContext.Message = messageTestI[theContext.Lang]
		theContext.AlertLevel = WARNING
		theContext.Title = titleConfig[theContext.Lang]
		render(w, "configure", theContext)
	case RUNNING:
		//wrong state, the system must be stopped before
		theContext.Message = messageTestR[theContext.Lang]
		theContext.AlertLevel = DANGER
		theContext.Title = titleRun[theContext.Lang]
		render(w, "run", theContext)
	case CONFIGURED, STOPPED:
		//correct state, let's test the system, and then to experiment page

		//check state of the sensors and put it on stateOfSensors
		//put here the authentic test code
		//put here the authentic test code

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
		if theContext.SetAccelerometer {
			theContext.StateOfAccelerometer = READY
		} else {
			theContext.StateOfAccelerometer = DISSABLED
		}
		if theContext.SetGyroscope {
			theContext.StateOfGyroscope = READY
		} else {
			theContext.StateOfGyroscope = DISSABLED
		}
		// test done, shows the result

		theContext.Title = titleTest[theContext.Lang]
		//theContext.Message = "System configured and Tested. Ready to run."
		theContext.Message = messageTestCS[theContext.Lang]
		theContext.AlertLevel = SUCCESS
		log.Println(">>>", theContext)
		render(w, "test", theContext)
	}
}

//Run allows to run the experiments
func Run(w http.ResponseWriter, req *http.Request) {
	log.Println(">>>", req.URL)
	log.Println(">>>", theContext)

	switch theContext.State {
	case INIT:
		//wrong state, show experiment page
		theContext.Message = messageRunI[theContext.Lang]
		theContext.AlertLevel = DANGER
		theContext.Title = titleExperiment[theContext.Lang]
		render(w, "experiment", theContext)
	case RUNNING:
		// we already are in this State
		// only put a message, but don't touch the running process
		theContext.Message = messageRunR[theContext.Lang]
		theContext.AlertLevel = WARNING
		theContext.Title = titleRun[theContext.Lang]
		render(w, "run", theContext)
	case CONFIGURED, STOPPED:
		//correct states, do the running process

		dataFileName := filepath.Join(StaticRoot, DataFilePath, theContext.ConfigurationName+DataFileExtension)
		//detect if file exists
		_, err := os.Stat(dataFileName)
		//create datafile is not exists
		if os.IsNotExist(err) {
			//create file to write
			log.Println("Creating ", dataFileName)
			theContext.DataFile, err = os.Create(dataFileName)
			if err != nil {
				log.Println(err.Error())
			}
			statusLine := fmt.Sprintf("### %v Data Acquisition: %s \n\n", time.Now(), theContext.ConfigurationName)
			theContext.DataFile.WriteString(statusLine)
			//formatLine := fmt.Sprintf("### [Ard], localTime(us), trackerTime(us), sensorTime(us), distance(mm), accX(g), accY(g), accZ(g), gyrX(gr/s), gyrY(gr/s), gyrZ(gr/s) \n\n")
			formatLine := fmt.Sprintf("### [Ard]; localTime(us); sensorTime(us)")
			if theContext.SetTrackerM == ON {
				formatLine += fmt.Sprintf("; trackerTime(us)")
			}
			if theContext.SetDistance == ON {
				formatLine += fmt.Sprintf("; distance(mm)")
			}
			if theContext.SetAccelerometer == ON {
				formatLine += fmt.Sprintf("; accX(g)")
				formatLine += fmt.Sprintf("; accY(g)")
				formatLine += fmt.Sprintf("; accZ(g)")
			}
			if theContext.SetGyroscope == ON {
				formatLine += fmt.Sprintf("; gyrX(gr/s)")
				formatLine += fmt.Sprintf("; gyrY(gr/s)")
				formatLine += fmt.Sprintf("; gyrZ(gr/s)")
			}
			formatLine += fmt.Sprintf("\n\n")
			theContext.DataFile.WriteString(formatLine)
			// sets the new time0 only with a new scenery
			theContext.setTime0()
		} else {
			//open fle to append
			log.Println("Openning ", dataFileName)
			theContext.DataFile, err = os.OpenFile(dataFileName, os.O_RDWR|os.O_APPEND, 0644)
			if err != nil {
				log.Println(err.Error())
			}
		}

		// running process instruction here!
		// running process instruction here!

		//waitTillButtonPushed(buttonA)
		hwio.DigitalWrite(theOshi.statusLed, hwio.HIGH)
		log.Println("Beginning.....")

		//activate arduino
		setArduinoStateON()

		// launch the trackers

		log.Printf("There are %v goroutines", runtime.NumGoroutine())
		log.Printf("Launching the Gourutines")

		theContext.State = RUNNING

		go theContext.readFromArduino()
		log.Println("Started Arduino")
		if theContext.SetTrackerA == ON {
			go readTracker("A", theOshi.trackerA)
			log.Println("Started Tracker A")
		}
		if theContext.SetTrackerB == ON {
			go readTracker("B", theOshi.trackerB)
			log.Println("Started Tracker B")
		}
		if theContext.SetTrackerC == ON {
			go readTracker("C", theOshi.trackerC)
			log.Println("Started Tracker C")
		}
		if theContext.SetTrackerD == ON {
			go readTracker("D", theOshi.trackerD)
			log.Println("Started Tracker D")
		}
		log.Printf("There are %v goroutines", runtime.NumGoroutine())
		//defer close the file to STOP

		theContext.Message = messageRunCS[theContext.Lang]
		theContext.AlertLevel = SUCCESS
		theContext.Title = titleRun[theContext.Lang]
		//theContext.State = RUNNING
		render(w, "run", theContext)
	}
}

//Stop allows to stop the experiments
func Stop(w http.ResponseWriter, req *http.Request) {
	log.Println(">>>", req.URL)
	log.Println(">>>", theContext)

	switch theContext.State {
	case INIT, CONFIGURED:
		theContext.Message = messageStopIC[theContext.Lang]
		theContext.AlertLevel = DANGER
		theContext.Title = titleExperiment[theContext.Lang]
		render(w, "experiment", theContext)
	case STOPPED:
		// we already are in this State
		// only put a message, but don't touch the process
		theContext.Message = messageStopS[theContext.Lang]
		theContext.AlertLevel = WARNING
		theContext.Title = titleStop[theContext.Lang]
		render(w, "experiment", theContext)
	case RUNNING:
		//correct state, do the stop process
		// stop process instruction here!
		// stop process instruction here!
		// stop process instruction here!

		//stop gorutines??
		log.Printf("Stop Gourutines (no at this time)")
		log.Printf("There are %v goroutines", runtime.NumGoroutine())

		//swich off the status led in the raspi
		hwio.DigitalWrite(theOshi.statusLed, hwio.LOW)
		// close the GPIO pins
		//hwio.CloseAll()

		//stop the arduino from read sensor and sending data via BT
		setArduinoStateOFF()
		log.Printf("Set Arduino OFF")

		//close the file
		err := theContext.DataFile.Sync()
		if err != nil {
			log.Println(err.Error())
		}
		theContext.DataFile.Close()

		theContext.Message = messageStopR[theContext.Lang]
		theContext.Title = titleStop[theContext.Lang]
		theContext.State = STOPPED
		theContext.AlertLevel = SUCCESS
		render(w, "stop", theContext)

	}
}

//Collect the data gathered in the experiments
func Collect(w http.ResponseWriter, req *http.Request) {
	log.Println(">>>", req.URL)
	log.Println(">>>", theContext)

	switch theContext.State {
	case INIT, CONFIGURED, STOPPED:
		//read the data directory and offers the files to be downloaded
		theContext.DataFiles, _ = filepath.Glob(filepath.Join(StaticRoot, DataFilePath, "*"+DataFileExtension))
		//log.Println(">>>> " + filepath.Join(StaticRoot, DataFilePath, "*"+DataExtension))
		//let only the file name, eliminate the path
		for i, f := range theContext.DataFiles {
			theContext.DataFiles[i] = path.Base(f)
		}

		log.Println(theContext.DataFiles)

		theContext.Title = titleCollect[theContext.Lang]
		if len(theContext.DataFiles) == 0 {
			theContext.Message = messageCollectICS0[theContext.Lang]
			theContext.AlertLevel = WARNING
		} else {
			theContext.Message = messageCollectICS[theContext.Lang]
			theContext.AlertLevel = INFO
		}
		render(w, "collect", theContext)
	case RUNNING:
		theContext.Message = messageCollectR[theContext.Lang]
		theContext.AlertLevel = WARNING
		theContext.Title = titleRun[theContext.Lang]
		render(w, "run", theContext)
	}

}

//Poweroff the system
func Poweroff(w http.ResponseWriter, req *http.Request) {
	log.Println(">>>", req.URL)
	log.Println(">>>", theContext)

	switch theContext.State {
	case INIT, CONFIGURED, STOPPED:
		// correct states
		if req.Method == "GET" {
			theContext.Message = messagePoweroffICSGet[theContext.Lang]
			theContext.AlertLevel = DANGER
			theContext.Title = titlePoweroff[theContext.Lang]
			render(w, "poweroff", theContext)
		} else { // POST
			log.Println("POST")
			req.ParseForm()
			log.Println(req.Form)
			if req.Form.Get("poweroff") == "YES" {
				//if YES, switch off the platform
				theContext.State = POWEROFF
				theContext.ConfigurationName = ""
				//message of poweroff state
				theContext.Message = messagePoweroffICSPostYes[theContext.Lang]
				theContext.AlertLevel = SUCCESS
				theContext.Title = titleTheEnd[theContext.Lang]
				render(w, "end", theContext)
				//wait some time to show the end page
				//time.Sleep(3 * time.Second)
				//halt the system
				log.Println("Poweroff!")
				// shutdown!!
				// shutdown!!
				// shutdown!!
				defer shutdown()
			} else {
				//message of initial state
				theContext.Message = messagePoweroffICSPostNo[theContext.Lang]
				theContext.AlertLevel = WARNING
				//initiated or not, shows the experiment page
				theContext.Title = titleExperiment[theContext.Lang]
				render(w, "experiment", theContext)
			}
		}
	case RUNNING:
		// wrong state
		theContext.Message = messagePoweroffR[theContext.Lang]
		theContext.AlertLevel = DANGER
		theContext.Title = titleRun[theContext.Lang]
		render(w, "run", theContext)
	}

}

func shutdown() {
	cmd := exec.Command("shutdown", "-h", "now")
	//cmd := exec.Command("shutdown", "-k", "now")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	} else { //command was successful
		log.Println("Bye!")
	}
}

//About shows the page with info
func About(w http.ResponseWriter, req *http.Request) {
	log.Println(">>>", req.URL)
	log.Println(">>>", theContext)

	theContext.Title = titleAbout[theContext.Lang]
	render(w, "about", theContext)
}

//Help shows information about the tool
func Help(w http.ResponseWriter, req *http.Request) {
	log.Println(">>>", req.URL)
	log.Println(">>>", theContext)

	theContext.Title = titleHelp[theContext.Lang]
	render(w, "help", theContext)
}

// render
func render(w http.ResponseWriter, tmpl string, cntxt Context) {
	log.Println("[render]>>>", cntxt)
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
	http.HandleFunc("/poweroff/", Poweroff)
	//http.HandleFunc("/end/", End)
	http.HandleFunc("/about/", About)
	http.HandleFunc("/help/", Help)
	http.HandleFunc(StaticURL, StaticHandler)

	// change this to show the real ip address of eth0
	//log.Println("Listening on 192.168.1.1:8000")

	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

	// close the GPIO pins
	defer theContext.SerialPort.Close()
	hwio.CloseAll()
}
