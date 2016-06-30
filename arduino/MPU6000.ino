
//Receive a read command to MPU6000

byte readRegister(int cs, byte thisRegister) {
  byte dato;
  SPI.beginTransaction(settingsHIGH);
    digitalWrite(cs, LOW);
      SPI.transfer(thisRegister | 0x80); // | 0x80 activa el MSB para indicar lectura.
      dato = SPI.transfer(0x00);         // Segundo byte de transferencia hacia el dispositivo.
    digitalWrite(cs, HIGH);
  SPI.endTransaction();  // take the chip select high to de-select:
  return dato;
}

//Sends a write command to MPU6000

void writeRegister(int cs, byte thisRegister, byte thisValue) {
  // take the chip select low to select the device:
  SPI.beginTransaction(settingsLOW);
    digitalWrite(cs, LOW);
      SPI.transfer(thisRegister); //Send register location
      SPI.transfer(thisValue);  //Send value to record into register
    digitalWrite(cs, HIGH);
  SPI.endTransaction();  // take the chip select high to de-select:
}

// Resetea un MPU6000 segun p. 41 del Manual de Registros, para el caso SPI
// demora 200ms
void resetMPU(int cs) {
  SPI.beginTransaction(settingsLOW);
    digitalWrite(cs, LOW);
      SPI.transfer(PWR_MGMT_1);
      SPI.transfer(PWR_MGMT_INIT);
    digitalWrite(cs, HIGH);
  SPI.endTransaction();

  delay(100); // 100ms para reiniciar el dispositivo

  // una posibilidad es SIGNAL_PATH_RESET con DEF_SIGNAL_PATH_RESET,
  // pero esta otra tambien es posible:
    
  SPI.beginTransaction(settingsLOW);
    digitalWrite(cs, LOW);
      SPI.transfer(USER_CTRL);
      SPI.transfer(DEF_USER_CTRL_I2C_IF_DIS | DEF_USER_CTRL_SIG_COND_RESET);
    digitalWrite(cs, HIGH);
  SPI.endTransaction();

  delay(100);
}

// Sleep a sensor
void sleepMPU(int cs) {
     SPI.beginTransaction(settingsLOW);
      digitalWrite(cs, LOW);
        SPI.transfer(PWR_MGMT_1);
        SPI.transfer(PWR_MGMT_SLEEP);
      digitalWrite(cs, HIGH);
     SPI.endTransaction();
     delay(50);
}

// Wakeup sensors with gyroz+temp_dis
void wakeupMPU(int cs) {
     SPI.beginTransaction(settingsLOW);
      digitalWrite(cs, LOW);
        SPI.transfer(PWR_MGMT_1);
        SPI.transfer(PWR_MGMT_WAKEUP | PWR_MGMT_DISTEMP | PWR_MGMT_CLK_SEL_PPLGYROZ);
      digitalWrite(cs, HIGH);
     SPI.endTransaction();
     delay(100);
}

/* Init a sensor */
void initMPU(byte ss) {
   Serial.print(ss,HEX);

   resetMPU(ss) ;   // hay dos retardos de 100 ms incluidos

   wakeupMPU(ss) ;  // hay un retardo de 100 ms incluido
   
   // indica la velocidad de muestreo de sensores.
   // en este caso
   SPI.beginTransaction(settingsLOW);
    digitalWrite(ss, LOW);
      SPI.transfer(SMPRT_DIV);     // hay DLPF.
      SPI.transfer(DEF_SMPRT_DIV); // colocamos la f muestreo en 100 Hz
    digitalWrite(ss, HIGH);
   SPI.endTransaction();

   delay(10);

   SPI.beginTransaction(settingsLOW);
    digitalWrite(ss, LOW);
      SPI.transfer(CONFIG); // no hay DLPF
      SPI.transfer(DEF_CONFIG_DLPF_CFG_44HZ);
    digitalWrite(ss, HIGH);
   SPI.endTransaction();
   
   delay(10);
   
   SPI.beginTransaction(settingsLOW);
    digitalWrite(ss, LOW);
      SPI.transfer(GYRO_CONFIG); // no hay prueba
      SPI.transfer(DEF_GYRO_CONFIG_1000);
    digitalWrite(ss, HIGH);
   SPI.endTransaction();

   delay(10);
   
   SPI.beginTransaction(settingsLOW);
    digitalWrite(ss, LOW);
      SPI.transfer(ACCEL_CONFIG); // no hay prueba
      SPI.transfer(DEF_ACCEL_CONFIG_4);
    digitalWrite(ss, HIGH);
   SPI.endTransaction();

   delay(10);
}

/* Init a sensor */
void MiniinitMPU(byte ss) {
  SPI.beginTransaction(settingsLOW);
    digitalWrite(ss, LOW);
      SPI.transfer(PWR_MGMT_1);
      SPI.transfer(PWR_MGMT_INIT);
    digitalWrite(ss, HIGH);
  SPI.endTransaction();

  delay(300); // 200ms para reiniciar el dispositivo

  SPI.beginTransaction(settingsLOW);
    digitalWrite(ss, LOW);
      SPI.transfer(USER_CTRL);
      SPI.transfer(DEF_USER_CTRL_I2C_IF_DIS);
    digitalWrite(ss, HIGH);
  SPI.endTransaction();

  delay(100);

  wakeupMPU(ss);
  
  SPI.beginTransaction(settingsLOW);
    digitalWrite(ss, LOW);
      SPI.transfer(ACCEL_CONFIG); // no hay prueba
      SPI.transfer(DEF_ACCEL_CONFIG_4);
    digitalWrite(ss, HIGH);
  SPI.endTransaction();


  delay(100);
}

