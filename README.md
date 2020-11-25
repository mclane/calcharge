# calcharge

calcharge is an addon to andig's [evcc](https://github.com/andig/evcc).

With calcharge you can use a standard ical calendar in order to plan charging sessions for your electric vehicle.  

## Function  
Within a configurable time interval, calcharge contacts the ical server and tries to load events within the next 24 hours. A calendar
event needs to have a summary field of the form "SoC xx%", xx being the desired state of charge at the start time of the event. calcharge also subscribes 
to the MQTT broker where evcc publishes its data. calcharge subscribes to socCharge which represents the actual state of charge of the vehicle. 
The difference between target and actual soc multiplied with the battery capacity gives the energy which needs to be
still charged to reach the desired target soc. The remaining charge time in evcc mode "now" (maximum power) is calculated. This time interval is subtracted
from the desired event start time and gives the "point of no return" time. If this time is still in the future, the evcc charging mode is unchanged. 
As soon as this point appears in the past, the evcc charge mode is changed to "now".  
With every new value of soc, the point of no return time is adjusted using the new actual soc

## Installation
As evcc, calcharge is written in Go, so a go compiler is required. Clone repository and build. Change config.yaml according to your needs. 

## Remarks
calcharge is still work in progress. Since I am new to Go, the code is certainly not very elegant; please provide hints for improvements.