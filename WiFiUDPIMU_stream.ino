#include <WiFi.h>
#include <AsyncUDP.h>
#include <Wire.h>

#include <Adafruit_MPU6050.h>
#include <Adafruit_Sensor.h>

const char * ssid = "linksys";
const char * pass = "";

const int NUM_AD_CHANNELS =5;
const int analogInputs[NUM_AD_CHANNELS] = {36,39,34,35,32};
int analogValues[NUM_AD_CHANNELS];
int analogValuesmV[NUM_AD_CHANNELS];
int timestamp = 0;

const int MPU_ADDR = 0x68; // I2C address of the MPU-6050. If AD0 pin is set to HIGH, the I2C address will be 0x69.

int16_t accelerometer_x, accelerometer_y, accelerometer_z; // variables for accelerometer raw data
int16_t gyro_x, gyro_y, gyro_z; // variables for gyro raw data
int16_t temperature; // variables for temperature data

char tmp_str[7]; // temporary variable used in convert function


AsyncUDP upd;

char* convert_int16_to_str(int16_t i) { // converts int16 to string. Moreover, resulting strings will have the same length in the debug monitor.
  sprintf(tmp_str, "%6d", i);
  return tmp_str;
}

void setup() {
  // put your setup code here, to run once:
  for (int i=0; i> NUM_AD_CHANNELS; i++){
    pinMode(analogInputs[i],INPUT);
  }
  //set the resolution to 12 bits (0-4096)
  analogReadResolution(12);

  //setup accelerometer
  Wire.begin();
  Wire.beginTransmission(MPU_ADDR); // Begins a transmission to the I2C slave (GY-521 board)
  Wire.write(0x6B); // PWR_MGMT_1 register
  Wire.write(0); // set to zero (wakes up the MPU-6050)
  //Wire.endTransmission(false);
  //Wire.beginTransmission(MPU_ADDR); // Begins a transmission to the I2C slave (GY-521 board)
  //Wire.write(0x19);//sample rate address
  //Wire.write(0);//set sample rate divider, higher will cause slower sample rate
  //Wire.write(0x1A);//Low pass filter address
  //Wire.write(0);//set to highest possible frequency
  Wire.endTransmission(true);

  Serial.begin(115200);
  WiFi.mode(WIFI_STA);
  WiFi.begin(ssid,pass);
  if (WiFi.waitForConnectResult() != WL_CONNECTED){
    Serial.println("Wifi Failed");
    while(1) {
       delay(500);
    }
  }
}

void updateIMUValues() {

  //read IMU Values
  Wire.beginTransmission(MPU_ADDR);
  Wire.write(0x3B); // starting with register 0x3B (ACCEL_XOUT_H) [MPU-6000 and MPU-6050 Register Map and Descriptions Revision 4.2, p.40]
  Wire.endTransmission(false); // the parameter indicates that the Arduino will send a restart. As a result, the connection is kept active.
  Wire.requestFrom(MPU_ADDR, 7*2, true); // request a total of 7*2=14 registers
  
  // "Wire.read()<<8 | Wire.read();" means two registers are read and stored in the same variable
  accelerometer_x = Wire.read()<<8 | Wire.read(); // reading registers: 0x3B (ACCEL_XOUT_H) and 0x3C (ACCEL_XOUT_L)
  accelerometer_y = Wire.read()<<8 | Wire.read(); // reading registers: 0x3D (ACCEL_YOUT_H) and 0x3E (ACCEL_YOUT_L)
  accelerometer_z = Wire.read()<<8 | Wire.read(); // reading registers: 0x3F (ACCEL_ZOUT_H) and 0x40 (ACCEL_ZOUT_L)
  temperature = Wire.read()<<8 | Wire.read(); // reading registers: 0x41 (TEMP_OUT_H) and 0x42 (TEMP_OUT_L)
  gyro_x = Wire.read()<<8 | Wire.read(); // reading registers: 0x43 (GYRO_XOUT_H) and 0x44 (GYRO_XOUT_L)
  gyro_y = Wire.read()<<8 | Wire.read(); // reading registers: 0x45 (GYRO_YOUT_H) and 0x46 (GYRO_YOUT_L)
  gyro_z = Wire.read()<<8 | Wire.read(); // reading registers: 0x47 (GYRO_ZOUT_H) and 0x48 (GYRO_ZOUT_L)
  timestamp = millis();

}

void updateAnalogValues(){
    for (int i = 0; i <NUM_AD_CHANNELS; i++){
    analogValues[i] = analogRead(analogInputs[i]);
    analogValuesmV[i] = analogReadMilliVolts(analogInputs[i]);
      // read the analog / millivolts value for pin 2:

  }

}

void sendAnalogValues() {
  String builtString = String("");

  for (int i =0; i < NUM_AD_CHANNELS; i++){
    builtString += String((int)analogValuesmV[i]);
    builtString += " mV";
    if (i < NUM_AD_CHANNELS -1){
      builtString += String("|");
    }
  }

  builtString += "|  ";
  builtString += String((int)timestamp);
  builtString += " ms since last boot";

  Serial.println(builtString);
  upd.broadcastTo(builtString.c_str(),2255);
  free(&builtString);
}

void sendIMUValues() {
  String builtString = String("");
  
  builtString += "aX = ";
  builtString += convert_int16_to_str(accelerometer_x);
  builtString += " | aY = ";
  builtString += convert_int16_to_str(accelerometer_y);
  builtString += " | aZ = ";
  builtString += convert_int16_to_str(accelerometer_z);
  builtString += " | gX = ";
  builtString += convert_int16_to_str(gyro_x);
  builtString += " | gY = ";
  builtString += convert_int16_to_str(gyro_y);
  builtString += " | gZ = ";
  builtString += convert_int16_to_str(gyro_z);
  builtString += " | Sensor Temperature = ";
  builtString += String((float)temperature/340.00+36.53);
  builtString += "|  ";
  builtString += String((int)timestamp);
  builtString += " ms since last boot";


  Serial.println(builtString);
  upd.broadcastTo(builtString.c_str(),2255);
  free(&builtString);

}


void sendAllValues(){
    String builtString = String("");
  
  builtString += "aX=";
  builtString += convert_int16_to_str(accelerometer_x);
  builtString += "|aY=";
  builtString += convert_int16_to_str(accelerometer_y);
  builtString += "|aZ=";
  builtString += convert_int16_to_str(accelerometer_z);
  builtString += "|gX=";
  builtString += convert_int16_to_str(gyro_x);
  builtString += "|gY=";
  builtString += convert_int16_to_str(gyro_y);
  builtString += "|gZ=";
  builtString += convert_int16_to_str(gyro_z);
  builtString += "|Temperature=";
  builtString += String((float)temperature/340.00+36.53);
  builtString += "|";
  builtString += String((int)timestamp);
  builtString += "ms since last boot|";

  for (int i =0; i < NUM_AD_CHANNELS; i++){
    builtString += String((int)analogValuesmV[i]);
    builtString += " mV";
    if (i < NUM_AD_CHANNELS -1){
      builtString += String("|");
    }
  }

  Serial.println(builtString);
  upd.broadcastTo(builtString.c_str(),2255);
  free(&builtString);

}

void loop() {
  // put your main code here, to run repeatedly:
  
  updateAnalogValues();
  //sendAnalogValues();
  updateIMUValues();
  //sendIMUValues();
  sendAllValues();
  //delay(10);

}
