/*
 MPU 6000 - 6 axis IMU - Breakout from Drotek - v2
 
 Shows the output of Accelerometer and Gyroscope Sensors, using
 SPI library. For details on the sensor, see:

 MPU6000 de InvenSense
 breakout de DroTek
 
 created 23 october 2014
 by Cesar Llamas
 */

// the sensor communicates using SPI, so include the library:

//SPISettings settingsLOW(  800000, MSBFIRST, SPI_MODE0); // SPI lento para configuracion 800KHz
//SPISettings settingsHIGH(8000000, MSBFIRST, SPI_MODE0); // SPI rapido para lectura      8MHz
SPISettings settingsLOW( 1000000, MSBFIRST, SPI_MODE0); // SPI lento para configuracion 800KHz
SPISettings settingsHIGH(1000000, MSBFIRST, SPI_MODE0); // SPI rapido para lectura      8MHz



const byte ACCEL_CONFIG = 0x1C;  // Configuracion y test del acelerometro
   const byte DEF_ACCEL_CONFIG_ST=0xE0; // 111-00-000: XA_ST | YA_ST | ZA_ST (2g al reset)
const byte DEF_ACCEL_CONFIG_2 = 0x00;   // 000-00-000: AFS_SEL(00=>2g)
const byte DEF_ACCEL_CONFIG_4 = 0x08;   // 000-01-000: AFS_SEL(01=>4g)
const byte DEF_ACCEL_CONFIG_8 = 0x10;   // 000-10-000: AFS_SEL(10=>8g)
const byte DEF_ACCEL_CONFIG_16= 0x18;   // 000-11-000: AFS_SEL(11=>16g)
                                  // res: 16384 LSB/g, 8192 LSB/g, 4096 LSB/g, 2048 LSB/g
   const byte DEF_ACCEL_CONFIG[4] = {DEF_ACCEL_CONFIG_2, DEF_ACCEL_CONFIG_4,
                                     DEF_ACCEL_CONFIG_8, DEF_ACCEL_CONFIG_16};                                    
   int accMode = 1; // +-4g
   
                                        
const byte GYRO_CONFIG = 0x1B;   // Configuracion y test del giroscopo
   const byte DEF_GYRO_CONFIG_ST =0xE0; // 111-00-000: XG_ST | YG_ST | ZG_ST | (250 º/s al reset)
const byte DEF_GYRO_CONFIG_250 = 0x00;  // 000-00-000: FS_SEL(00=>250 º/s)
const byte DEF_GYRO_CONFIG_500 = 0x08;  // 000-01-000: FS_SEL(01=>500 º/s)
const byte DEF_GYRO_CONFIG_1000= 0x10;  // 000-10-000: FS_SEL(10=>1000 º/s)
const byte DEF_GYRO_CONFIG_2000= 0x18;  // 000-11-000: FS_SEL(11=>2000 º/s)
                                   // res: 131 LSB/w, 65,5 LSB/w, 32,8 LSB/w, 16,4 LSB/w
                                   // con w en º/s.
   const byte DEF_GYRO_CONFIG[4] = {DEF_GYRO_CONFIG_250, DEF_GYRO_CONFIG_500,
                                    DEF_GYRO_CONFIG_1000, DEF_GYRO_CONFIG_2000};
   int gyrMode = 2; //+-1000 DPS
   
const byte SIGNAL_PATH_RESET=0x68;// Reset de la circuiteria de los sensores internos
   const byte DEF_SIGNAL_PATH_RESET=B00000111;// 0x07=0000 0111: GYRO_RESET | ACCE__RESET | TEMP_RESET
   
const byte USER_CTRL = 0x6A;      // control de registros FIFO y  I2C/SPI
    const byte DEF_USER_CTRL_I2C_IF_DIS     = B00010000; // 0x10; I2C Disable
    const byte DEF_USER_CTRL_SIG_COND_RESET = B00000001; // clear signal path y sensor registers
    // mola usar los dos flags porque hace el signal PATH reset que pide la pagina 41 del manual reg.

// vamos a muestrear a 100 Hz con un filtro pasabaja digital que no sirve de mucho a 94 Hz.
// esto quiere decir que la frecuencia del giro de 1kHz se divide entre 10, que nos da un
// (SMPRD_DIV+1) = 10, con lo que SMPRD = 0x09.
//
const byte SMPRT_DIV = 0x19;     // Factor de Frecuencia de muestreo
   byte DEF_SMPRT_DIV = 0x09;    // 100 Hz (1kHz/(1+0x09)=100Hz)
                                 // con filtro pasa baja, es decir:
//Configuracion del DLPF que podemos poner a 0 o para 100HZ de SR a 188.
const byte CONFIG = 0x1A;
   const byte DEF_CONFIG_DLPF_CFG_94HZ = 0x02; // B00-000-010
   const byte DEF_CONFIG_DLPF_CFG_44HZ = 0x03; // B00-000-011
   const byte DEF_CONFIG_DLPF_0FF = 0x00;
                                 


const byte PWR_MGMT_1 = 0x6B;     // Gestion de energia y reset.
   const byte PWR_MGMT_INIT    = B10000000; //0x80=1000 0000 = DEVICE_RESET
   const byte PWR_MGMT_SLEEP   = B01000000; //0x40=0100 0000 = SLEEP
   const byte PWR_MGMT_WAKEUP  = B00000000; //0x00 despierta el MPU
   const byte PWR_MGMT_DISTEMP = B00001000; //0x04 = deshabilita temp.
   const byte PWR_MGMT_CLK_SEL_PPLGYROZ=B00000011; //0x03 = gyroz como reloj interno
   // un despertar tipico es algo as como:
   // PWR_MGMT_WAKEUP | PWR_MGMT_DISTEMP | PWR_MGMT_CLK_SEL_PPLGYROZ
   
// registros de lectura del acelerometro
const byte ACCEL_XOUTH = 0x3B;   // Valor alto del eje x del acel.
const byte ACCEL_XOUTL = 0x3C;   // Valor bajo del eje x del acel.
const byte ACCEL_YOUTH = 0x3D;   // Valor alto del eje y del acel.
const byte ACCEL_YOUTL = 0x3E;   // Valor bajo del eje y del acel.
const byte ACCEL_ZOUTH = 0x3F;   // Valor alto del eje z del acel.
const byte ACCEL_ZOUTL = 0x40;   // Valor bajo del eje z del acel.

// registros de escritura del giroscopo
const byte GYRO_XOUTH = 0x43;   // Valor alto del eje x del giro.
const byte GYRO_XOUTL = 0x44;   // Valor bajo del eje x del giro.
const byte GYRO_YOUTH = 0x45;   // Valor alto del eje y del giro.
const byte GYRO_YOUTL = 0x46;   // Valor bajo del eje y del giro.
const byte GYRO_ZOUTH = 0x47;   // Valor alto del eje z del giro.
const byte GYRO_ZOUTL = 0x48;   // Valor bajo del eje z del giro.

// lee un registro de una MPU6000 utilizando acceso SPI
byte readRegister(int chipselectpin, byte thisRegister);

// escribe un registro de una MPU6000 utilizando acceso SPI
void writeRegister(int cs, byte thisRegister, byte thisValue);

// Resetea un MPU6000 segun p. 41 del Manual de Registros, para el caso SPI
// demora 200ms
void resetMPU(int cs);

// Sleep a sensor
void sleepMPU(int cs);

// Wakeup sensors with gyroz+temp_dis
void wakeupMPU(int cs);

// inicializa una MPU por su chipselect
void initMPU(byte ss);

// inicializacin minima de una MPU por su chipselect
void miniinitMPU(byte ss);


