package printer

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	goserial "go.bug.st/serial"
)

type Client struct {
	mu            sync.Mutex
	port          goserial.Port
	readStop      chan struct{}
	logListeners  []func(string)
	tempListeners []func(hCurrent, hTarget, bCurrent, bTarget string)
	tempBuf       string
	monitoring    bool
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) AddLogListener(f func(string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logListeners = append(c.logListeners, f)
}

func (c *Client) AddTempListener(f func(hCurrent, hTarget, bCurrent, bTarget string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tempListeners = append(c.tempListeners, f)
}

func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.port != nil
}

func (c *Client) Connect(portName string, baud int) error {
	mode := &goserial.Mode{BaudRate: baud}
	port, err := goserial.Open(portName, mode)
	if err != nil {
		return err
	}
	if err := port.SetReadTimeout(500 * time.Millisecond); err != nil {
		port.Close()
		return err
	}

	c.mu.Lock()
	if c.port != nil {
		c.mu.Unlock()
		port.Close()
		return fmt.Errorf("already connected")
	}
	stop := make(chan struct{})
	c.port = port
	c.readStop = stop
	c.mu.Unlock()

	go c.readLoop(port, stop)
	return nil
}

func (c *Client) Disconnect() error {
	c.mu.Lock()
	if c.port == nil {
		c.mu.Unlock()
		return nil
	}
	if c.readStop != nil {
		close(c.readStop)
	}
	port := c.port
	c.port = nil
	c.readStop = nil
	c.mu.Unlock()
	return port.Close()
}

func (c *Client) SendRaw(cmd string) error {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}
	c.mu.Lock()
	port := c.port
	c.mu.Unlock()
	if port == nil {
		return fmt.Errorf("not connected")
	}
	payload := cmd + "\n"
	_, err := port.Write([]byte(payload))
	return err
}

// Operations
func (c *Client) ResetZOffset() error {
	for _, cmd := range []string{"M851 Z0", "G28", "G0 Z0"} {
		if err := c.SendRaw(cmd); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) MoveToZ(z float64) error {
	return c.SendRaw(fmt.Sprintf("G0 Z%.3f", z))
}

func (c *Client) ApplyZOffset(z float64) error {
	return c.SendRaw(fmt.Sprintf("M851 Z%.3f", z))
}

func (c *Client) SaveSettings() error {
	return c.SendRaw("M500")
}

func (c *Client) StartTempMonitoring() error {
	c.monitoring = true
	return c.SendRaw("M155 S1")
}

func (c *Client) StopTempMonitoring() error {
	c.monitoring = false
	return c.SendRaw("M155 S0")
}

func (c *Client) PreheatHotend(temp float64) error {
	return c.SendRaw(fmt.Sprintf("M104 S%.0f", temp))
}

func (c *Client) PreheatBed(temp float64) error {
	return c.SendRaw(fmt.Sprintf("M140 S%.0f", temp))
}

// internal
func (c *Client) readLoop(port goserial.Port, stop <-chan struct{}) {
	buf := make([]byte, 1024)
	reHot := regexp.MustCompile(`T:([0-9.]+)\s*/\s*([0-9.]+)`)
	reBed := regexp.MustCompile(`B:([0-9.]+)\s*/\s*([0-9.]+)`)
	for {
		select {
		case <-stop:
			return
		default:
		}

		n, err := port.Read(buf)
		if err != nil {
			if isTimeoutError(err) {
				continue
			}
			c.broadcastLog(fmt.Sprintf("Read error: %v\n", err))
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if n == 0 {
			continue
		}
		data := string(buf[:n])
		c.broadcastLog(data)
		c.consumeTempData(data, reHot, reBed)
	}
}

func (c *Client) broadcastLog(text string) {
	c.mu.Lock()
	listeners := append([]func(string){}, c.logListeners...)
	c.mu.Unlock()
	for _, f := range listeners {
		f(text)
	}
}

func (c *Client) broadcastTemp(hCurrent, hTarget, bCurrent, bTarget string) {
	c.mu.Lock()
	listeners := append([]func(string, string, string, string){}, c.tempListeners...)
	c.mu.Unlock()
	for _, f := range listeners {
		f(hCurrent, hTarget, bCurrent, bTarget)
	}
}

func (c *Client) consumeTempData(chunk string, reHot, reBed *regexp.Regexp) {
	c.tempBuf += chunk
	lines := strings.Split(c.tempBuf, "\n")
	c.tempBuf = lines[len(lines)-1]
	if !c.monitoring {
		return
	}
	for _, line := range lines[:len(lines)-1] {
		var hcur, htar, bcur, btar string
		if m := reHot.FindStringSubmatch(line); len(m) == 3 {
			hcur, htar = m[1], m[2]
		}
		if m := reBed.FindStringSubmatch(line); len(m) == 3 {
			bcur, btar = m[1], m[2]
		}
		if hcur != "" || bcur != "" {
			c.broadcastTemp(hcur, htar, bcur, btar)
		}
	}
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	type timeout interface{ Timeout() bool }
	if t, ok := err.(timeout); ok && t.Timeout() {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "timeout")
}
