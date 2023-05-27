import sys
import socket
import re
import matplotlib.pyplot as plt
import numpy

client = socket.socket(socket.AF_INET, socket.SOCK_DGRAM,socket.IPPROTO_UDP)

client.setblocking(1)

client.bind(('',2255))

datatags=['aX','aY','aZ','gX','gY','gZ','Temperature','ms since last boot']
datasets=[[],[],[],[],[],[],[],[]]
offsets=[0,0,0,0,0,0,0]
scales=[1,1,1,1,1,1,1]
scaling=1


def check_data():
    try:
        data, addr = client.recvfrom(1024)
        #print("recieved message: %s from %s" % (data,addr))
        return data.decode()
    except socket.error:
        print('socket error')
        return 
    

def fetchdata(datalength):

    while len(datasets[0])<datalength:

        line = check_data()

        if line:

            split_line = line.split('|')
            isaccel = re.search('aX',line)            
            if isaccel:
                for tag,dataset,splt in zip(datatags,datasets,split_line):
                    #print(splt)
                    result = re.search(tag,splt)
                    numresult = re.search('[-+]?([0-9]*\.[0-9]+|[0-9]+)',splt)
                    numberdata = numresult.group(0)
                    numberdata = float(numberdata)
                    if result:
                        dataset.append(numberdata)
                            
                
def plotdata():
  
    datasets[7]=numpy.subtract(datasets[7],datasets[7][0])
    datasets[7]=numpy.divide(datasets[7],1e3)
    datasets[0:3]=numpy.divide(datasets[0:3],scaling)

    for i in range(3,6):
        datasets[i]=numpy.multiply(datasets[i],90/12500)
        datasets[i]=numpy.cumsum(numpy.multiply(datasets[i],numpy.mean(numpy.diff(datasets[7]))))

    for i in range(len(datatags)-2):
        plt.subplot(2,3,i+1)
        plt.plot(datasets[7],datasets[i])
        plt.xlabel('time')
        plt.ylabel(datatags[i])

    plt.show()

        
    
print('Acquiring Offset, please wait')
fetchdata(2e2)

for i in range(0,3):
    scales[i]=numpy.mean(datasets[i])
scaling=numpy.sqrt(numpy.sum(numpy.square(scales[0:3])))/9.8
    
for i in range(3,6):
    offsets[i]=numpy.mean(datasets[i])
print('offset complete')
print(scaling)

fetchdata(7e2)



for i in range(3,6):
    datasets[i]=numpy.subtract(datasets[i],offsets[i])


plotdata()
    



##re.search('[-+]?([0-9]*\.[0-9]+|[0-9]+)','aX =   4588 ')
