#include <SPI.h>

#include "MPU6000.h"

// state values
const int stateON = 1;
const int stateOFF = 0;


// conector SPI utilizado:
// MPU6000 sensor attached to pins:
//   D13 : SCLK
//   D12 : SDO
//   D11 : SDI     // este y los anteriores son estandar para SPI.h
//   D07 : /CS
byte imuPin = 7; // chip select del IMU
//   VCC a 5V y GND como le corresponda.

// MPU6000 sensibility
// Accelerometer
const float acc2G  = 16384.0;
const float acc4G  =  8192.0;
const float acc8G  =  4096.0;
const float acc16G =  2048.0;
// Gyro
const float gyro250os  = 131.0;
const float gyro500os  =  65.5;
const float gyro1000os =  32.8;
const float gyro2000os =  16.4;
// Temp
// C = temp/340+36.53


// connect BT module TX to D0
// connect BT module RX to D1
// connect BT Vcc to 3,3V, GND to GND

// tracker
byte trackerPin = 3;

// boton físico
byte buttonPin = 2;
boolean buttonChangedState = false; // indicating if the button changed the state
int buttonState;            // the current reading of the input pin
int lastButtonState = LOW;  // the previous reading of the input pin
long lastDebounceTime = 0;  // the last time the output pin was toggled
long debounceDelay = 50;    // the debounce time; incresase is the output flickers

// led de estado
byte ledPin = 6;
int ledState;

char command; // guarda el caracter proveniente del Serial.

volatile unsigned long tiempo, tiempoANT=0;

// estado: 0 - Off, 1 - On
volatile int state = stateOFF; // innitial state

// Ultrasonidos
int pinEcho = 9;
int pinTrigger = 8;

// Tracker
int pinTracker = 5;

// outputLine
volatile boolean sincro = false;
//String line="#";   // line with the message, composed with data from time and data from sensors.
                   // #time us,distance mm, Accx g, Accy g, Accz g, Gyrox º/s, Gyroy º/s, Gyroz º/s, Temp ºC
                   // if starts with @ means a sincronization point detected by the tracker

//
// binary data output line
//

typedef struct {
  char firstChar;
  unsigned long theTime;
  unsigned long distance;
  float          accX;
  float          accY;
  float          accZ;
  float          gyrX;
  float          gyrY;
  float          gyrZ;
  char lastChar;
} OutputLine;

volatile OutputLine outputLine;
int bytesOutputLine; // size of the outputLine after being scaped and in bytes
volatile byte outputLineInBytes[sizeof(OutputLine)];



///////
// functions
//////

void scapeAndBytes() {
  // tamaño minimo de la trama de tiempo
  bytesOutputLine=sizeof(OutputLine); 

  byte *i = (byte *) &outputLine;
  byte *o = (byte *) outputLineInBytes;
  
  //*o++ = 0x7E;  // inicio trama tiempo

  for (int j=0; j<sizeof(OutputLine); j++, i++) {
    /*
    switch (*i) {
      case 0x7D: *o++ = 0x7D; *o++ = 0x00; bytesOutputLine++; break;
    // suponemos que no hay perdidas ni inserciones
    //case 0x7E:  es otro inicio de trama y solo esperamos el final
    //case 0x23:  es inicio trama de datos y solo esperamos el final
      case 0x24: *o++ = 0x7D; *o++ = 0x03; bytesOutputLine++; break;
      case 0x11: *o++ = 0x7D; *o++ = 0x04; bytesOutputLine++; break;
      case 0x13: *o++ = 0x7D; *o++ = 0x05; bytesOutputLine++; break;
      default  : *o++ = *i;
    }
    */
    *o++ = *i;
  }

  //*o = 0x24;    // endding mark '$'
}


String aTexto() {
  String linea = "";
  linea += outputLine.firstChar;
  linea += ", ";
  linea += outputLine.theTime;
  linea += ", ";
  linea += outputLine.distance;
  linea += ", ";
  linea += outputLine.accX;
  linea += ", ";
  linea += outputLine.accY;
  linea += ", ";
  linea += outputLine.accZ;
  linea += ", ";
  linea += outputLine.gyrX;
  linea += ", ";
  linea += outputLine.gyrY;
  linea += ", ";
  linea += outputLine.gyrZ;
  linea += ", ";
  linea += outputLine.lastChar;

  return linea;
}


boolean buttonPressed(){// read the state of the switch into a local variable:
  int reading = digitalRead(buttonPin);
  // check to see if you just pressed the button 
  // (i.e. the input went from LOW to HIGH),  and you've waited 
  // long enough since the last press to ignore any noise:  

  // If the switch changed, due to noise or pressing:
  if (reading != buttonState) {
    // reset the debouncing timer
    lastDebounceTime = millis();
  }
 
  if ((millis() - lastDebounceTime) > debounceDelay) {
    // whatever the reading is at, it's been there for longer
    // than the debounce delay, so take it as the actual current state:

    // if the button state has changed:       
    if (reading != lastButtonState) {
 
      buttonState = reading;

      //only toggle the state if the new button state is HIGH
      if (buttonState == HIGH){
        buttonChangedState = !buttonChangedState;
      }
    }
  }
  return buttonChangedState;
}


String status()
{
  String myStatus="[";
  myStatus+="OK"; // set by default, TODO check the status
  myStatus+="] ";
  switch (state) {
    case stateON:
      myStatus+="On";
      break;
    case stateOFF:
      myStatus+= "Off";
      break;
    default: 
      myStatus+= "Undefined";
      break;
  }
   // add more status info just in case
  
  return myStatus;
}


///////////////////////
// Interrupts
///////////////////////

void changeState() {
  if (state == stateON){
    state = stateOFF;
    digitalWrite(ledPin, LOW);
  } else {
    state = stateON;
    digitalWrite(ledPin, HIGH);
  }
}

void sendSincro(){
  //String outputline="@";  
  //Serial.write("@\n");
  //line.setCharAt(0,'@');
  sincro = true;
  //Serial.write("~");
}

//////////////////////////////////////////////////////////////////////

void setup()  
{
  Serial.begin(9600);
  
  SPI.begin();
  
  delay(100);

  pinMode(imuPin, OUTPUT);
  MiniinitMPU(imuPin);
  
  // para los settings de SPI ver MPU6000.h: SPISettings.
  delay(1000);
  
  Serial.begin(9600);
  // Send test message to other device
  Serial.println("Hola, me llamo Arduino, y soy tu nuevo vecino");
  
  pinMode(ledPin, OUTPUT); // led
  
  pinMode(buttonPin, INPUT); // pushbutton
  attachInterrupt(digitalPinToInterrupt(buttonPin),changeState, FALLING);
  
  pinMode(trackerPin, INPUT); // tracker
  attachInterrupt(digitalPinToInterrupt(trackerPin),sendSincro, RISING);

  // state related values
  lastButtonState = LOW;
  state = stateOFF;

  // ultrasonic sensor
  pinMode(pinEcho, INPUT);  // conexion del pinEcho
  pinMode(pinTrigger, OUTPUT); // conexion del pinTrigger
  inicia_SR(pinEcho,pinTrigger);

  Serial.begin(9600);  // just in case
}



void loop() 
{
  
  unsigned long distancia;
  
  short int xacc=0x0, yacc=0x0, zacc=0x0;
  //float fxacc, fyacc, fzacc;
  byte xaccl, xacch;
  byte yaccl, yacch;
  byte zaccl, zacch;
  
  short int temp=0x0;
  byte templ, temph;

  short int xgir=0x0, ygir=0x0, zgir=0x0;
  //float fxgir, fygir, fzgir;
  byte xgirl, xgirh;
  byte ygirl, ygirh;
  byte zgirl, zgirh;

  
  if (Serial.available())
  // if text arrived in from BT serial...
  // s or S -> Status
  // n or N -> On
  // f or F -> Off
  {
    command=(Serial.read());
    switch (command) {
    case 's': //case 'S':
      digitalWrite(ledPin, HIGH);
      Serial.println("Status ...");
      Serial.println(status());
      break;
    case 'n': //case 'N':
      digitalWrite(ledPin, HIGH);
      Serial.println("Readding ...");
      state = stateON;
      break;
    case 'f': //case 'F':
      digitalWrite(ledPin, LOW);
      Serial.println("Stopping ...");
      state = stateOFF;
      break;
    default:
      Serial.println("Send 'n' to set readdings ON");
      Serial.println("Send 'f' to set readdings OFF");
      break;
    }   
  }

  
  
  if (state == stateON) { // in case of state ON, make the readdings and send by Serial
    
    if (sincro) { 
      //line = "@"; //first caracter of the output in the case of sincro by tracker
      outputLine.firstChar=0x64;  // @
      sincro = false; // @ set, reset sincro value
    }
    else {
      //line = "#"; //first caracter of the output, except in the case of sincro by tracker
      outputLine.firstChar=0x23;  // #
    }
    outputLine.lastChar=0x24; // $
    tiempo = micros();
    //line+= tiempo;
    outputLine.theTime=micros(); 
    tiempoANT=tiempo;
    //line +=", ";
     
    //distancia = distanciaPulso();
    outputLine.distance=distanciaPulso();
    //line += distancia;
    //line +=", ";  
     
    SPI.beginTransaction(settingsHIGH); // definido en MPU6000.h
    digitalWrite(imuPin, LOW);
    SPI.transfer(ACCEL_XOUTH | 0x80); // see "readRegister()"
    xacch = SPI.transfer(0x00);
    xaccl = SPI.transfer(0x00);
    yacch = SPI.transfer(0x00);
    yaccl = SPI.transfer(0x00);
    zacch = SPI.transfer(0x00);
    zaccl = SPI.transfer(0x00);
    
    temph = SPI.transfer(0x00);
    templ = SPI.transfer(0x00); 

    xgirh = SPI.transfer(0x00);
    xgirl = SPI.transfer(0x00);
    ygirh = SPI.transfer(0x00);
    ygirl = SPI.transfer(0x00);
    zgirh = SPI.transfer(0x00);
    zgirl = SPI.transfer(0x00);
        
    digitalWrite(imuPin, HIGH); // cortar SPI
    SPI.endTransaction();
    
    // ACCELEROMETER
    // la aceleracion viene en multiplos de "g" (9.8 m/(s*s))
    //
    xacc = xacch; xacc = xacc << 8;   xacc = xacc | xaccl;
    outputLine.accX = xacc / acc4G;
    yacc = yacch; yacc = yacc << 8;   yacc = yacc | yaccl;
    outputLine.accY = yacc / acc4G;
    zacc = zacch; zacc = zacc << 8;   zacc = zacc | zaccl;
    outputLine.accZ = zacc / acc4G;
        
    // GYROSCPE
    // la velocidad angular viene dada en grados/s
    //
    xgir = xgirh; xgir = xgir << 8;   xgir = xgir | xgirl;
    outputLine.gyrX = xgir / gyro250os;
    ygir = ygirh; ygir = ygir << 8;   ygir = ygir | ygirl;
    outputLine.gyrY = ygir / gyro250os;
    zgir = zgirh; zgir = zgir << 8;   zgir = zgir | zgirl;
    outputLine.gyrZ = zgir / gyro250os;
    
    // TEMP
    // the temp is in F
    //temp = temph; temp = temp << 8;   temp = temp | templ;
    //line += temp / 340 + 36.53;
        
    //Serial.println(aTexto());
    scapeAndBytes();
    Serial.write((byte *) outputLineInBytes, sizeof(outputLineInBytes));
  }
  //delay(100);
}
