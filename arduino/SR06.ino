int _pEcho;
int _pTrigger;


void inicia_SR(int pinEcho, int pinTrigger) {
  extern int _pEcho;
  extern int _pTrigger;

  _pEcho = pinEcho;
  _pTrigger = pinTrigger;
}

unsigned long distanciaPulso() {
  extern int _pEcho;
  extern int _pTrigger;

  long duracion;  // vamos a guardar microsegundos
  
  digitalWrite(_pTrigger, LOW);

  delayMicroseconds(5); // he visto valores desde 2 hasta 5.
  digitalWrite(_pTrigger, HIGH);

  // la unidad espera un pulso que sirve de disparo para que
  // comience el calculo de la distancia.
  delayMicroseconds(10); // ancho minimo que pide la unidad HC-SR04
  digitalWrite(_pTrigger, LOW);
  // nos devuelve un pulso cuya anchura es el tiempo (us) que demora
  // en volver el echo de un tren de pulsos a 40 kHz.
  // en resumen, nos da la distancia en formato tiempo, que tenemos
  // que alimentar a una formula para calcular la distancia.
  duracion = pulseIn(_pEcho,HIGH)/5.8; // en mm
  
  return duracion;
}

