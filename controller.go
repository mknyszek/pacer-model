package main

type ControllerConfig struct {
	Kp     float64 `json:"k_p"`
	Ti     float64 `json:"t_i"`
	Tt     float64 `json:"t_t"`
	Period float64 `json:"period"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
}

type Controller struct {
	ControllerConfig
	integral float64
}

func NewController(cfg ControllerConfig) *Controller {
	return &Controller{ControllerConfig: cfg}
}

func (s *Controller) output(input, setpoint float64) (rawOutput, output float64) {
	prop := s.Kp * (setpoint - input)
	rawOutput = prop + s.integral
	output = rawOutput
	if output < s.Min {
		output = s.Min
	} else if output > s.Max {
		output = s.Max
	}
	return rawOutput, output
}

func (s *Controller) update(input, setpoint, rawOutput, output float64) {
	if s.Ti != 0 && s.Tt != 0 {
		s.integral += (s.Kp*s.Period/s.Ti)*(setpoint-input) + (s.Period/s.Tt)*(output-rawOutput)
	}
}

func (s *Controller) Next(input, setpoint float64) float64 {
	rawOutput, output := s.output(input, setpoint)
	s.update(input, setpoint, rawOutput, output)
	return output
}
