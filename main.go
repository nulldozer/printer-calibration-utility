package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/andlabs/ui"
	_ "github.com/andlabs/ui/winmanifest"
	"go.bug.st/serial"

	"github.com/nulldozer/printer-calibration-utility/printer"
)

type serialUI struct {
	window        *ui.Window
	tab           *ui.Tab
	mainBox       *ui.Box
	connectionBox *ui.Box
	portDropdown  *ui.Combobox
	baudDropdown  *ui.Combobox
	connectBtn    *ui.Button
	statusLabel   *ui.Label
	client        *printer.Client
	serialTabUI   *serialTab
	zTabUI        *zOffsetTab
	tempTabUI     *tempTab
	bedTabUI      *bedLevelTab

	ports     []string
	baudRates []int

	mu sync.Mutex
}

func main() {
	ui.Main(func() {
		app := &serialUI{
			baudRates: []int{250000, 115200, 57600, 38400, 19200, 9600},
			client:    printer.NewClient(),
		}
		app.buildUI()
	})
}

func (s *serialUI) buildUI() {
	s.window = ui.NewWindow("Printer Calibration Utility", 800, 600, true)
	s.window.OnClosing(func(*ui.Window) bool {
		s.disconnect()
		ui.Quit()
		return true
	})
	ui.OnShouldQuit(func() bool {
		s.window.Destroy()
		return true
	})

	mainBox := ui.NewVerticalBox()
	mainBox.SetPadded(true)
	s.window.SetChild(mainBox)
	s.window.SetMargined(true)
	s.mainBox = mainBox

	s.statusLabel = ui.NewLabel("Disconnected")
	mainBox.Append(s.statusLabel, false)

	s.connectionBox = ui.NewVerticalBox()
	s.connectionBox.SetPadded(false)
	s.connectionBox.Append(s.makeConnectionGrid(""), false)
	mainBox.Append(s.connectionBox, false)

	s.tab = ui.NewTab()
	s.serialTabUI = newSerialTab(s.client)
	s.tab.Append("Serial Monitor", s.serialTabUI.Build())
	s.tab.SetMargined(0, true)
	s.zTabUI = newZOffsetTab(s.client)
	s.tab.Append("Z Offset", s.zTabUI.Build())
	s.tab.SetMargined(1, true)
	s.tempTabUI = newTempTab(s.client)
	s.tab.Append("Temperature", s.tempTabUI.Build())
	s.tab.SetMargined(2, true)
	s.bedTabUI = newBedLevelTab(s.client)
	s.tab.Append("Bed Leveling", s.bedTabUI.Build())
	s.tab.SetMargined(3, true)
	mainBox.Append(s.tab, true)

	s.refreshPorts()
	s.window.Show()
}

func (s *serialUI) makeConnectionGrid(selectedPort string) ui.Control {
	grid := ui.NewGrid()
	grid.SetPadded(true)

	s.portDropdown = ui.NewCombobox()
	grid.Append(ui.NewLabel("Serial Port"), 0, 0, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
	grid.Append(s.portDropdown, 1, 0, 1, 1, true, ui.AlignFill, false, ui.AlignFill)
	targetIndex := -1
	for i, port := range s.ports {
		s.portDropdown.Append(port)
		if selectedPort != "" && port == selectedPort {
			targetIndex = i
		}
	}
	if targetIndex >= 0 {
		s.portDropdown.SetSelected(targetIndex)
	} else if len(s.ports) > 0 {
		s.portDropdown.SetSelected(0)
	}

	s.baudDropdown = ui.NewCombobox()
	for _, baud := range s.baudRates {
		s.baudDropdown.Append(fmt.Sprintf("%d", baud))
	}
	if s.baudDropdown.Selected() == -1 {
		s.baudDropdown.SetSelected(0)
	}
	grid.Append(ui.NewLabel("Baud Rate"), 0, 1, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
	grid.Append(s.baudDropdown, 1, 1, 1, 1, true, ui.AlignFill, false, ui.AlignFill)

	s.connectBtn = ui.NewButton("Connect")
	s.connectBtn.OnClicked(func(*ui.Button) {
		if s.isConnected() {
			s.disconnect()
			return
		}
		s.connect()
	})
	grid.Append(s.connectBtn, 2, 0, 1, 1, false, ui.AlignFill, false, ui.AlignFill)

	refresh := ui.NewButton("Refresh")
	refresh.OnClicked(func(*ui.Button) {
		s.refreshPorts()
	})
	grid.Append(refresh, 2, 1, 1, 1, false, ui.AlignFill, false, ui.AlignFill)

	return grid
}

func (s *serialUI) refreshPorts() {
	current := s.selectedPort()
	ports, err := serial.GetPortsList()
	if err != nil {
		s.appendLog(fmt.Sprintf("Failed to list ports: %v", err))
		s.setStatus("Unable to list ports")
		return
	}
	if len(ports) == 0 {
		s.appendLog("No serial ports found")
		s.setStatus("No serial ports found")
	}

	s.ports = ports
	if s.connectionBox != nil {
		// rebuild the grid to refresh dropdown items
		s.connectionBox.Delete(0)
		s.connectionBox.Append(s.makeConnectionGrid(current), false)
	}
}

func (s *serialUI) connect() {
	portName := s.selectedPort()
	if portName == "" {
		s.appendLog("Select a serial port before connecting")
		return
	}
	baud := s.selectedBaud()
	if baud == 0 {
		s.appendLog("Select a baud rate before connecting")
		return
	}

	if err := s.client.Connect(portName, baud); err != nil {
		s.appendLog(fmt.Sprintf("Failed to open %s: %v", portName, err))
		return
	}

	s.updateConnectionUI(true)
	s.appendLog(fmt.Sprintf("Connected to %s @ %d baud", portName, baud))
	s.setStatus(fmt.Sprintf("Connected to %s @ %d baud", portName, baud))
}

func (s *serialUI) disconnect() {
	_ = s.client.Disconnect()

	s.updateConnectionUI(false)
	s.appendLog("Disconnected")
	s.setStatus("Disconnected")
}

func (s *serialUI) selectedPort() string {
	idx := s.portDropdown.Selected()
	if idx < 0 || idx >= len(s.ports) {
		return ""
	}
	return s.ports[idx]
}

func (s *serialUI) selectedBaud() int {
	idx := s.baudDropdown.Selected()
	if idx < 0 || idx >= len(s.baudRates) {
		return 0
	}
	return s.baudRates[idx]
}

func (s *serialUI) isConnected() bool {
	return s.client.IsConnected()
}

func (s *serialUI) updateConnectionUI(connected bool) {
	ui.QueueMain(func() {
		if connected {
			s.connectBtn.SetText("Disconnect")
			s.portDropdown.Disable()
			s.baudDropdown.Disable()
		} else {
			s.connectBtn.SetText("Connect")
			s.portDropdown.Enable()
			s.baudDropdown.Enable()
		}
	})
	if s.serialTabUI != nil {
		s.serialTabUI.OnConnectionChanged(connected)
	}
	if s.zTabUI != nil {
		s.zTabUI.OnConnectionChanged(connected)
	}
	if s.tempTabUI != nil {
		s.tempTabUI.OnConnectionChanged(connected)
	}
	if s.bedTabUI != nil {
		s.bedTabUI.OnConnectionChanged(connected)
	}
}

func (s *serialUI) appendLog(text string) {
	s.appendToLogs(text, false)
}

func (s *serialUI) appendLogCommand(text string) {
	s.appendToLogs(text, true)
}

func (s *serialUI) appendToLogs(text string, addNewline bool) {
	if addNewline && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	ui.QueueMain(func() {
		if s.serialTabUI != nil && s.serialTabUI.log != nil {
			s.serialTabUI.log.Append(text)
		}
	})
}

func (s *serialUI) setStatus(text string) {
	ui.QueueMain(func() {
		s.statusLabel.SetText(text)
	})
}

func (s *serialUI) clearInput() {
	ui.QueueMain(func() {
		if s.serialTabUI != nil && s.serialTabUI.inputEntry != nil {
			s.serialTabUI.inputEntry.SetText("")
		}
	})
}
