package controller

type Controller interface {
	Next(input, setpoint float64) float64
}
