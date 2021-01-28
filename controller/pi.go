package controller

type PI struct {
	PIConfig
	integral float64
}

type PIConfig struct {
	Kp     float64 `json:"k_p"`
	Ti     float64 `json:"t_i"`
	Tt     float64 `json:"t_t"`
	Period float64 `json:"period"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
}

func NewPI(cfg *PIConfig) *PI {
	return &PI{PIConfig: *cfg}
}

func (c *PI) output(input, setpoint float64) (rawOutput, output float64) {
	prop := c.Kp * (setpoint - input)
	rawOutput = prop + c.integral
	output = rawOutput
	if output < c.Min {
		output = c.Min
	} else if output > c.Max {
		output = c.Max
	}
	return rawOutput, output
}

func (c *PI) update(input, setpoint, rawOutput, output float64) {
	if c.Ti != 0 && c.Tt != 0 {
		c.integral += (c.Kp*c.Period/c.Ti)*(setpoint-input) + (c.Period/c.Tt)*(output-rawOutput)
	}
}

func (c *PI) Next(input, setpoint float64) float64 {
	rawOutput, output := c.output(input, setpoint)
	c.update(input, setpoint, rawOutput, output)
	return output
}
