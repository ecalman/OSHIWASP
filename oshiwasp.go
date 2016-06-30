package main

import (
    "github.com/mrmorphic/hwio"
    "fmt"
    "time"
    "os"
    "net/http"
    "regexp"
    "errors"
    "html/template"
    "log"
    "runtime"
    "github.com/tarm/serial"
    "bytes"
    "encoding/binary"
    "bufio"
    "io"
)

const (

    // web related
    tmplPath = "tmpl/" // path of the template files .html in the local file system
    dataPath = "data/" // path of the data files in the local file system
    dataFileExtension = ".csv" //  data file extension in the local file system

    // arduino serial comm related
    commDevName = "/dev/rfcomm1"  //name of the BT device
    bauds = 9600 // bauds of the BT serial channel

    // setup of the pinout in the raspberry
    statusLedPin = "gpio7" // green      
    actionLedPin = "gpio8"  // yellow     

    buttonAPin = "gpio24"    // start
    buttonBPin = "gpio23"    // stop

    trackerAPin = "gpio22"    
    trackerBPin = "gpio18"    
    trackerCPin = "gpio17"    
    trackerDPin = "gpio4"    

    //States for the acquisition
    //                  resume <- PAUSED <- pause
    //                     |                 ^
    //                     |                 |
    //                     +----+        +---+
    //                           \      /
    //0 -- new -> NEW -- start -> RUNNING -- stop -> STOPPED
    //             ^       ^                           |
    //             |       +------- start -------------+
    //             +--------------- new ---------------+
    //
    stateNEW = "NEW"
    stateRUNNING = "RUNNING"
    statePAUSED = "PAUSED"
    stateSTOPPED = "STOPPED"
    stateERROR = "ERROR"

    // sensor configuration states
    // ON activated
    ON = true

    // OFF deactivated
    OFF = false

)


var (

    c chan int //channel initialitation
    //actionLed hwio.Pin // indicating action in the system

    templates = template.Must(template.ParseGlob(tmplPath+"*.tmpl"))
    validPath = regexp.MustCompile("^/(index|new|status|start|pause|resume|stop|download|data)/([a-zA-Z0-9]+)$")

    theSensorData SensorData
    theSensorDataInBytes SensorDataInBytes

    theAcq=new(Acquisition)
    theOshi=new(Oshiwasp)

)

//AAAAAAAAAAAAAA
// Acquisition section
//AAAAAAAAAAAAAA


// SensorData data for sensors in Arduino
type SensorData struct{
        sincroMicroSeconds uint32
        sensorMicroSeconds uint32
        distance uint32
        accX float32
        accY float32
        accZ float32
        gyrX float32
        gyrY float32
        gyrZ float32
}
// SensorDataIn Bytes data for sensors in Arduino in bytes
type SensorDataInBytes struct{
    	sincroMicroSecondsInBytes []byte
	sensorMicroSecondsInBytes []byte
	distanceInBytes []byte
	accXInBytes []byte
	accYInBytes []byte
	accZInBytes []byte
	gyrXInBytes []byte
	gyrYInBytes []byte
	gyrZInBytes []byte
}

//Acquisition definition
type Acquisition struct{
    outputFileName string
    outputFile *os.File
    state string
    time0 time.Time
    //arduino
    serialPort *serial.Port
    //configuration
    ConfigurationName string
    TrackerA          bool
    TrackerB          bool
    TrackerC          bool
    TrackerD          bool
    TrackerM          bool
    Distance          bool
    AccGyro           bool
}

// StateOfSensors state
type StateOfSensors struct { // state of the sensors
        ConfigurationName string
        TrackerA          string
        TrackerB          string
        TrackerC          string
        TrackerD          string
        TrackerM          string
        Distance          string
        AccGyro           string
}


func (acq *Acquisition) connectArduinoSerialBT(){
    // config the comm port for serial via BT
    commPort := &serial.Config{Name: commDevName, Baud: bauds}
    // open the serial comm with the arduino via BT
    acq.serialPort, _ = serial.OpenPort(commPort)
    //defer acq.serialPort.Close()
    log.Printf("Open serial device %s", commDevName)
}

func (acq *Acquisition) setArduinoStateON(){
// activate the readdings in Arduino sending 'ON'
	log.Printf("before write on")
        _, err := acq.serialPort.Write([]byte("n"))
	log.Printf("after write on")
        if err != nil {
                log.Fatal(err)
        }
}

func (acq *Acquisition) setArduinoStateOFF(){
// deactivate the readdings in Artudino sending 'OFF'
	log.Printf("before write off")
        _, err := acq.serialPort.Write([]byte("f"))
	log.Printf("after write off")
        if err != nil {
	        log.Printf("error!! after write off")
                log.Fatal(err)
        }
}


func (acq *Acquisition) setTime0(){
    acq.time0 = time.Now()
}

func (acq *Acquisition) getTime0() time.Time {
    return acq.time0 
}

func (acq *Acquisition) getState() string{
    return acq.state
}

func (acq *Acquisition) setState(s string){
    acq.state = s
    log.Printf("State set to %s\n", acq.state)
}

func (acq *Acquisition) setStateNEW(){
    acq.state = stateNEW
    acq.setArduinoStateOFF()

    log.Printf("State: NEW")
}

func (acq *Acquisition) setStateRUNNING(){
    acq.state = stateRUNNING
    acq.setArduinoStateON()
    log.Printf("State: RUNNING")
}

func (acq *Acquisition) setStatePAUSED(){
    acq.state = statePAUSED
    acq.setArduinoStateOFF()
    log.Printf("State: PAUSED")
}

func (acq *Acquisition) setStateSTOPPED(){
    acq.state = stateSTOPPED
    acq.setArduinoStateOFF()
    log.Printf("State: STOPPED")
}

func (acq *Acquisition) setStateERROR(){
    acq.state = stateERROR
    acq.setArduinoStateOFF()
    log.Printf("State: ERROR")
}

// set the output file name based in the configurationName
func (acq *Acquisition) setOutputFileName(s string){
    acq.outputFileName = s
    log.Printf("Output Filename set to %s\n", acq.outputFileName)
}

func (acq *Acquisition) createOutputFile(){
    var e error
    acq.setOutputFileName(dataPath + acq.ConfigurationName + dataFileExtension)
    acq.outputFile, e = os.Create(acq.outputFileName)
    if e != nil {
        panic(e)
    }
    statusLine := fmt.Sprintf("### %v Data Acquisition: %s \n\n", time.Now(),acq.ConfigurationName)
    acq.outputFile.WriteString(statusLine)
    formatLine := fmt.Sprintf("### [Ard], localTime(us), sincroTime(us), sensorTime(us), distance(mm), accX(g), accY(g), accZ(g), gyrX(gr/s), gyrY(gr/s), gyrZ(gr/s) \n\n")
    acq.outputFile.WriteString(formatLine)
    
    log.Printf("Cretated output File %s", acq.outputFileName)
}

func (acq *Acquisition) reopenOutputFile(){
    var e error
    acq.outputFile, e = os.OpenFile(acq.outputFileName,os.O_WRONLY|os.O_APPEND, 0666)
    if e != nil {
        panic(e)
    }
    log.Printf("Reopen output File %s", acq.outputFileName)
}

func (acq Acquisition) closeOutputFile(){ //close the output file 
    acq.outputFile.Close()
    log.Printf("Closed output File %s", acq.outputFileName)
}


func (acq *Acquisition) initiate(){
    //acq.setOutputFileName(dataPath+dataFileName+dataFileExtension)
    //acq.createOutputFile()
    acq.connectArduinoSerialBT()
    log.Printf("Arduino connected!")
    acq.setStateNEW()
}

//OOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO
// Oshiwasp section: Raspberry sensors
//OOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO

// Oshiwasp definition of configuration of raspberry sensors, leds and buttons
type Oshiwasp struct {
    statusLed hwio.Pin
    actionLed hwio.Pin
    buttonA hwio.Pin
    buttonB hwio.Pin
    trackerA hwio.Pin
    trackerB hwio.Pin
    trackerC hwio.Pin
    trackerD hwio.Pin
}




func (oshi *Oshiwasp) initiate(){

    var e error
    // Set up 'trakers' as inputs
    oshi.trackerA, e = hwio.GetPinWithMode(trackerAPin, hwio.INPUT)
    if e != nil {
        panic(e)
    }
    log.Printf("Set pin %s as trackerA\n",trackerAPin)

    oshi.trackerB, e = hwio.GetPinWithMode(trackerBPin, hwio.INPUT)
    if e != nil {
        panic(e)
    }
    log.Printf("Set pin %s as trackerB\n",trackerBPin)

    oshi.trackerC, e = hwio.GetPinWithMode(trackerCPin, hwio.INPUT)
    if e != nil {
        panic(e)
    }
    log.Printf("Set pin %s as trackerC\n",trackerCPin)

    oshi.trackerD, e = hwio.GetPinWithMode(trackerDPin, hwio.INPUT)
    if e != nil {
        panic(e)
    }
    log.Printf("Set pin %s as trackerD\n",trackerDPin)

 
    // Set up 'buttons' as inputs
    oshi.buttonA, e = hwio.GetPinWithMode(buttonAPin, hwio.INPUT)
    if e != nil {
        panic(e)
    }
    log.Printf("Set pin %s as buttonA\n",buttonAPin)

    oshi.buttonB, e = hwio.GetPinWithMode(buttonBPin, hwio.INPUT)
    if e != nil {
        panic(e)
    }
    log.Printf("Set pin %s as buttonB\n",buttonBPin)

    // Set up 'leds' as outputs
    oshi.statusLed, e = hwio.GetPinWithMode(statusLedPin, hwio.OUTPUT)
    if e != nil {
        panic(e)
    }
    log.Printf("Set pin %s as statusLed\n",statusLedPin)

    oshi.actionLed, e = hwio.GetPinWithMode(actionLedPin, hwio.OUTPUT)
    if e != nil {
        panic(e)
    }
    log.Printf("Set pin %s as actionLed\n",actionLedPin)
}


func readTracker(name string, trackerPin hwio.Pin){

    oldValue := 0 //value readed from tracker, initially set to 0, because the tracker was innactive
    timeAction := time.Now() // time of the action detected

    // loop
    for theAcq.getState() != stateSTOPPED {
           // Read the tracker value
           value, e := hwio.DigitalRead(trackerPin)
           if e != nil {
                panic(e)
           }
        //timeActionOld=timeAction //store the last time
        timeAction= time.Now() // time at this point
        // Did value change?
        if (value == 1) && (value != oldValue) {
	    if  theAcq.getState() != statePAUSED {
                dataString := fmt.Sprintf("[%s], %d,\n",
                              name,int64(timeAction.Sub(theAcq.getTime0())/time.Microsecond))
                log.Println(dataString)
                theAcq.outputFile.WriteString(dataString)
            }

            // Write the value to the led indicating somewhat is happened
            if (value == 1) {
                hwio.DigitalWrite(theOshi.actionLed, hwio.HIGH)
            } else {
                hwio.DigitalWrite(theOshi.actionLed, hwio.LOW)
            }
        }
        oldValue = value
    }
}

func (acq *Acquisition) readFromArduino(){

    var register, reg []byte
    // operate with the gobal variables theSensorData and theSensorDataInBytes; more speed?

    // don't use the first readding ??  I'm not sure about that
    reader := bufio.NewReader(acq.serialPort)
    // find the begging of an stream of data from the sensors
    register, err := reader.ReadBytes('\x24');
    if err != nil { log.Fatal(err) }
    //log.Println(register)
    //log.Printf(">>>>>>>>>>>>>>")

    // loop
    for acq.getState() != stateSTOPPED {
        // Read the serial and decode
        
        register = nil
        reg = nil

        //n, err = s.Read(register)
        for len(register) < 38 { // in case of \x24 chars repeted the length will be less than the expected 38 bytes
            reg, err = reader.ReadBytes('\x24');
            if err != nil { log.Fatal(err) }
            register = append(register, reg...)
        }

        receptionTime := time.Now() // time of the action detected

        if (register[0] == '\x23') { // if first byte is '#', lets decode the stream of bytes in register

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



  
	if  acq.getState() != statePAUSED {
	    //compound the dataline and write to the output
            //receptionTime= time.Now() // Alternative: time at this point 
            dataString := fmt.Sprintf("[%s], %d, %d, %d, %d, %f, %f, %f, %f, %f, %f\n",
                       "Ard", int64(receptionTime.Sub(theAcq.getTime0())/time.Microsecond),
                       theSensorData.sincroMicroSeconds, theSensorData.sensorMicroSeconds, theSensorData.distance, 
                       theSensorData.accX, theSensorData.accY, theSensorData.accZ,
                       theSensorData.gyrX, theSensorData.gyrY, theSensorData.gyrZ)

            log.Println(dataString)
            acq.outputFile.WriteString(dataString)
        }
        // Write the value to the led indicating somewhat is happened
        hwio.DigitalWrite(theOshi.actionLed, hwio.HIGH)
        hwio.DigitalWrite(theOshi.actionLed, hwio.LOW)
    }
}

// another version looking for more speed, based in local variables
func (acq *Acquisition) readFromArduino2(){

    var register, reg []byte
    var sincroMicroSecondsInBytes []byte
    var sensorMicroSecondsInBytes []byte
    var distanceInBytes []byte
    var accXInBytes []byte
    var accYInBytes []byte
    var accZInBytes []byte
    var gyrXInBytes []byte
    var gyrYInBytes []byte
    var gyrZInBytes []byte
    var sincroMicroSeconds uint32
    var sensorMicroSeconds uint32
    var distance uint32
    var accX float32
    var accY float32
    var accZ float32
    var gyrX float32
    var gyrY float32
    var gyrZ float32

    // don't use the first readding ??  I'm not sure about that
    reader := bufio.NewReader(acq.serialPort)
    // find the begging of an stream of data from the sensors
    register, err := reader.ReadBytes('\x24');
    if err != nil { log.Fatal(err) }
    //log.Println(register)
    //log.Printf(">>>>>>>>>>>>>>")

    // loop
    for acq.getState() != stateSTOPPED {
        // Read the serial and decode
        
        register = nil
        reg = nil

        //n, err = s.Read(register)
        for len(register) < 38 { // in case of \x24 chars repeted maked the length will be less than the expected 38 bytes
            reg, err = reader.ReadBytes('\x24');
            if err != nil { log.Fatal(err) }
            register = append(register, reg...)
        }

        receptionTime := time.Now() // time of the action detected

        if (register[0] == '\x23') {

            //decode the register

             sincroMicroSecondsInBytes = register[1:5]
             buf := bytes.NewReader(sincroMicroSecondsInBytes)
             err = binary.Read(buf, binary.LittleEndian, &sincroMicroSeconds)

             sensorMicroSecondsInBytes = register[5:9]
             buf = bytes.NewReader(sensorMicroSecondsInBytes)
             err = binary.Read(buf, binary.LittleEndian, &sensorMicroSeconds)

             distanceInBytes = register[9:13]
             buf = bytes.NewReader(distanceInBytes)
             err = binary.Read(buf, binary.LittleEndian, &distance)

             accXInBytes = register[13:17]
             buf = bytes.NewReader(accXInBytes)
             err = binary.Read(buf, binary.LittleEndian, &accX)

             accYInBytes = register[17:21]
             buf = bytes.NewReader(accYInBytes)
             err = binary.Read(buf, binary.LittleEndian, &accY)

             accZInBytes = register[21:25]
             buf = bytes.NewReader(accZInBytes)
             err = binary.Read(buf, binary.LittleEndian, &accZ)

             gyrXInBytes = register[25:29]
             buf = bytes.NewReader(gyrXInBytes)
             err = binary.Read(buf, binary.LittleEndian, &gyrX)

             gyrYInBytes = register[29:33]
             buf = bytes.NewReader(gyrYInBytes)
             err = binary.Read(buf, binary.LittleEndian, &gyrY)

             gyrZInBytes = register[33:37]
             buf = bytes.NewReader(gyrZInBytes)
             err = binary.Read(buf, binary.LittleEndian, &gyrZ)

        } // if



  
	if  acq.getState() != statePAUSED {
	    //compound the dataline and write to the output
            //receptionTime= time.Now() // Alternative: time at this point 
            dataString := fmt.Sprintf("[%s], %v, %d, %d, %d, %f, %f, %f, %f, %f, %f\n",
                       "Ard", receptionTime.Sub(theAcq.getTime0()),
                       sincroMicroSeconds, sensorMicroSeconds, distance, 
                       accX, accY, accZ, gyrX, gyrY, gyrZ)

            log.Println(dataString)
            acq.outputFile.WriteString(dataString)
        }
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


//WWWWWWWWWWWWW
// http section
//WWWWWWWWWWWWW

// Page title and body web page content
type Page struct {
    Title string
    Body string
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
    err := templates.ExecuteTemplate(w, tmpl+".html", p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func getTitle(w http.ResponseWriter, r *http.Request) (string, error) {
    m := validPath.FindStringSubmatch(r.URL.Path)
    if m == nil {
        http.NotFound(w, r)
        return "", errors.New("Invalid Page Name")
    }
    return m[2], nil //the name is the second subexpression
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
    log.Println(r.URL) //get the requested URL
    if r.Method == "GET" {
        theAcq.setStateSTOPPED()
        p := &Page{Title: "Main Page", Body: "Make an action. State:  "+theAcq.state}
        //renderTemplate(w,"index",p)
        t, _ := template.ParseFiles(tmplPath+"header.tmpl", tmplPath+"main.tmpl", tmplPath+"footer.tmpl")
        t.ExecuteTemplate(w, "header", p)
        t.ExecuteTemplate(w, "main", p)
        t.ExecuteTemplate(w, "footer", nil)
        t.Execute(w, nil)
     } else {
        //
     }

}

func newHandler(w http.ResponseWriter, r *http.Request) {
    log.Println(r.URL, r.Method) // get the requested URL
    if r.Method == "GET" {
        p := &Page{Title: "New configuration", Body: "Configure the sensors and put a name to it. State: " + theAcq.state}
        t, _ := template.ParseFiles(tmplPath+"header.tmpl", tmplPath+"new.tmpl", tmplPath+"footer.tmpl")
        t.ExecuteTemplate(w, "header", p)
        t.ExecuteTemplate(w, "new", p)
        t.ExecuteTemplate(w, "footer", nil)
        t.Execute(w, nil)
    } else { // POST, process the form and set the data about the new configuration
        r.ParseForm()
        // logic part of login
        //validation phase will be here
        //is valid, put the form data into the configuration struct
        theAcq.ConfigurationName = r.Form.Get("ConfigurationName")
        if r.Form.Get("TrackerA") == "on" {
            theAcq.TrackerA = ON
        } else {
            theAcq.TrackerA = OFF
        }
        if r.Form.Get("TrackerB") == "on" {
            theAcq.TrackerB = ON
        } else {
            theAcq.TrackerB = OFF
        }
        if r.Form.Get("TrackerC") == "on" {
            theAcq.TrackerC = ON
        } else {
            theAcq.TrackerC = OFF
        }
        if r.Form.Get("TrackerD") == "on" {
            theAcq.TrackerD = ON
        } else {
            theAcq.TrackerD = OFF
        }
        if r.Form.Get("TrackerM") == "on" {
            theAcq.TrackerM = ON
        } else {
            theAcq.TrackerM = OFF
        }
        if r.Form.Get("Distance") == "on" {
            theAcq.Distance = ON
        } else {
            theAcq.Distance = OFF
        }
        if r.Form.Get("AccGyro") == "on" {
            theAcq.AccGyro = ON
        } else {
            theAcq.AccGyro = OFF
        }
        //echo
        log.Println(r.Form)
        log.Println("ConfigurationName", r.Form.Get("ConfigurationName"))
        log.Println("TrackerA", r.Form["TrackerA"])
        log.Println("TrackerB", r.Form["TrackerB"])
        log.Println("TrackerC", r.Form["TrackerC"])
        log.Println("TrackerD", r.Form["TrackerD"])
        log.Println("TrackerM", r.Form["TrackerM"])
        log.Println("Distance", r.Form["Distance"])
        log.Println("AccGyro", r.Form["AccGyro"])
        log.Println("Configuration State:", theAcq)


	theAcq.createOutputFile()
        theAcq.setStateNEW()

        //once processed the form, reditect to tha index page
        http.Redirect(w, r, "/", http.StatusFound)
        }
}

func startHandler(w http.ResponseWriter, r *http.Request) {

    // manage file depending the previous state 
    if theAcq.getState() == stateSTOPPED {
        theAcq.reopenOutputFile() 
        log.Printf("Reopen file %s\n", theAcq.outputFile);
    }

    // sets the new time0 only with a new scenery
    if theAcq.getState() == stateNEW {
        theAcq.setTime0();
    }

    theAcq.setStateRUNNING()

    //waitTillButtonPushed(buttonA)
    p := &Page{Title: "Start", Body: "State: "+theAcq.state}
    hwio.DigitalWrite(theOshi.statusLed, hwio.HIGH)
    log.Println("Beginning.....");

    renderTemplate(w,"start",p)

    // launch the trackers

    log.Printf("There are %v goroutines", runtime.NumGoroutine())
    log.Printf("Launching the Gourutines")

    
    go theAcq.readFromArduino()
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
}

func pauseHandler(w http.ResponseWriter, r *http.Request) {

    theAcq.setStatePAUSED()

    p := &Page{Title: "Start", Body: "State: "+theAcq.state}
    hwio.DigitalWrite(theOshi.statusLed, hwio.HIGH)

    renderTemplate(w,"start",p)
}

func resumeHandler(w http.ResponseWriter, r *http.Request) {

    theAcq.setStateRUNNING()

    p := &Page{Title: "Start", Body: "State: "+theAcq.state}
    hwio.DigitalWrite(theOshi.statusLed, hwio.HIGH)

    renderTemplate(w,"start",p)
}

func stopHandler(w http.ResponseWriter, r *http.Request) {

    theAcq.setStateSTOPPED()
    hwio.DigitalWrite(theOshi.statusLed, hwio.LOW)
    log.Println("Finnishing.....");
    // close the GPIO pins
    //hwio.CloseAll()
    theAcq.closeOutputFile() //close the file when finished
    p := &Page{Title: "Stop", Body:"State: "+theAcq.state}
    renderTemplate(w,"stop",p)
    //log.Printf("There are %v goroutines", runtime.NumGoroutine())
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
    //http.ServeFile(w, r, theAcq.outputFileName)
    log.Println(r.URL, r.Method) // get the requested URL
    if r.Method == "GET" {
        p := &Page{Title: "Download", Body: theAcq.outputFileName}
        t, _ := template.ParseFiles(tmplPath+"header.tmpl", tmplPath+"download.tmpl", tmplPath+"footer.tmpl")
        t.ExecuteTemplate(w, "header", p)
        t.ExecuteTemplate(w, "download", p)
        t.ExecuteTemplate(w, "footer", nil)
        t.Execute(w, nil)
    } else { // POST, 
        //
    }
}

func getRequest(w http.ResponseWriter, r *http.Request){
    file_requested := "./"+r.URL.Path
    http.ServeFile(w, r, file_requested)
}


func staticHandler(w http.ResponseWriter, r *http.Request) {
    static_file := r.URL.Path[len("/"+dataPath):]
    if len(static_file) != 0 {
        f, err := http.Dir(dataPath).Open(static_file)
        if err == nil {
            content := io.ReadSeeker(f)
            http.ServeContent(w, r, static_file, time.Now(), content)
            return
        }
    }
    http.NotFound(w, r)
}

////////////
// main
////////////

func main() {

    // setup 
    mux := http.NewServeMux()
    mux.HandleFunc("/", indexHandler)
    mux.HandleFunc("/index", indexHandler)
    mux.HandleFunc("/new", newHandler)
    mux.HandleFunc("/start", startHandler)
    mux.HandleFunc("/pause", pauseHandler)
    mux.HandleFunc("/resume", resumeHandler)
    mux.HandleFunc("/stop", stopHandler)
    mux.HandleFunc("/download",downloadHandler)
    mux.HandleFunc(dataPath,staticHandler)

    theAcq.initiate();
    theOshi.initiate();


    // starting the web service...
    // http.Handle("/data", http.FileServer(http.Dir("./data")))
    log.Println("Listennig on http://localhost:8080/")
    log.Fatal(http.ListenAndServe(":8080", mux))

    log.Println("Closed http://localhost:8080/")
    // close the GPIO pins
    defer theAcq.serialPort.Close()
    hwio.CloseAll()
} 
